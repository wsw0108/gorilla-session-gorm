package gorm

import (
	"encoding/base32"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jinzhu/gorm"
)

type Store struct {
	db *gorm.DB

	tableName         string
	initTableDisabled bool
	gcDisabled        bool
	gcInterval        time.Duration

	keyPairs       [][]byte
	secureDisabled bool

	Options *sessions.Options

	codecs []securecookie.Codec
	ticker *time.Ticker
}

type sessionItem struct {
	ID        string    `gorm:"size:255;primary_key"`
	Data      string    `gorm:"size:4096"`
	ExpiredAt time.Time `gorm:"index"`
	CreatedAt time.Time
}

func NewStore(db *gorm.DB, options ...StoreOption) *Store {
	store := &Store{
		db:         db,
		tableName:  "sessions",
		gcInterval: 10 * time.Minute,
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 86400 * 30,
		},
	}

	for _, option := range options {
		option(store)
	}

	store.db = db.Table(store.tableName)

	if !store.initTableDisabled {
		store.db.AutoMigrate(&sessionItem{})
	}

	store.codecs = securecookie.CodecsFromPairs(store.keyPairs...)

	if !store.gcDisabled {
		store.ticker = time.NewTicker(store.gcInterval)
		go store.gc()
	}

	store.MaxAge(store.Options.MaxAge)

	return store
}

func (s *Store) Close() error {
	if !s.gcDisabled {
		s.ticker.Stop()
	}
	return nil
}

func (s *Store) gc() {
	for range s.ticker.C {
		s.clean()
	}
}

func (s *Store) clean() {
	now := gorm.NowFunc()
	query := "expired_at <= ?"
	var err error

	var count int64
	err = s.db.Where(query, now).Count(&count).Error
	if err != nil || count == 0 {
		if err != nil {
			log.Println(err.Error())
		}
		return
	}

	err = s.db.Unscoped().Where(query, now).Delete(&sessionItem{}).Error
	if err != nil {
		log.Println(err.Error())
	}
}

func (store *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(store, name)
}

func (store *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(store, name)
	options := *store.Options
	session.Options = &options
	session.IsNew = true
	if cookie, err := r.Cookie(name); err == nil {
		if err := securecookie.DecodeMulti(name, cookie.Value, &session.ID, store.codecs...); err != nil {
			return session, nil
		}
		item := &sessionItem{}
		if err := store.db.Where("id = ? AND expired_at > ?", session.ID, gorm.NowFunc()).First(item).Error; err != nil {
			return session, nil
		}
		if !store.secureDisabled {
			if err := securecookie.DecodeMulti(session.Name(), item.Data, &session.Values, store.codecs...); err != nil {
				return session, nil
			}
		} else {
			values := make(map[string]interface{})
			if err := json.Unmarshal([]byte(item.Data), &values); err != nil {
				return session, nil
			}
			for k, v := range values {
				session.Values[k] = v
			}
		}
		session.IsNew = false
	}
	return session, nil
}

func (store *Store) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	if session.Options.MaxAge < 0 {
		if session.ID != "" {
			if err := store.db.Delete(&sessionItem{ID: session.ID}).Error; err != nil {
				return err
			}
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	var err error
	var data string
	if !store.secureDisabled {
		data, err = securecookie.EncodeMulti(session.Name(), session.Values, store.codecs...)
	} else {
		var b []byte
		values := make(map[string]interface{})
		for k, v := range session.Values {
			kk := fmt.Sprint(k)
			values[kk] = v
		}
		b, err = json.Marshal(values)
		if err != nil {
			return err
		}
		data = string(b)
	}
	if err != nil {
		return err
	}

	now := gorm.NowFunc()
	expiredAt := now.Add(time.Second * time.Duration(session.Options.MaxAge))

	if session.ID == "" {
		session.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(
				securecookie.GenerateRandomKey(32)), "=")
		item := &sessionItem{
			ID:        session.ID,
			Data:      data,
			CreatedAt: now,
			ExpiredAt: expiredAt,
		}
		if err = store.db.Create(item).Error; err != nil {
			return err
		}
	} else {
		item := &sessionItem{
			ID:        session.ID,
			Data:      data,
			ExpiredAt: expiredAt,
		}
		if err = store.db.Save(item).Error; err != nil {
			return err
		}
	}

	cookieValue, err := securecookie.EncodeMulti(session.Name(), session.ID, store.codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, sessions.NewCookie(session.Name(), cookieValue, session.Options))

	return nil
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting
// Options.MaxAge = -1 for that session.
func (store *Store) MaxAge(age int) {
	store.Options.MaxAge = age
	for _, codec := range store.codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}
