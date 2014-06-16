// Package geoip provides a thin wrapper around libGeoIP for looking up
// geographical information about IP addresses.
package geoip

import (
	"runtime"
	"unsafe"
)

// #cgo LDFLAGS: -lGeoIP
// #include "GeoIP.h"
// #include "GeoIPCity.h"
import "C"

// CachingStrategy determines what data libGeoIP will cache.
type CachingStrategy int

const (
	// CacheDefault caches no data.
	CacheDefault CachingStrategy = C.GEOIP_STANDARD

	// CacheAll caches all data in memory.
	CacheAll CachingStrategy = C.GEOIP_MEMORY_CACHE

	// CacheMRU caches the most recently used data in memory.
	CacheMRU CachingStrategy = C.GEOIP_INDEX_CACHE
)

// Options are the set of options provided by libGeoIP.
type Options struct {
	Caching        CachingStrategy // Caching determines what data will be cached.
	ReloadOnUpdate bool            // ReloadOnUpdate will watch the data files for updates.
	UseMMap        bool            // UseMMap enables MMAP for the data files.
}

func (o Options) bitmask() int32 {
	v := int32(o.Caching)

	if o.ReloadOnUpdate {
		v |= C.GEOIP_CHECK_CACHE
	}

	if o.UseMMap {
		v |= C.GEOIP_MMAP_CACHE
	}

	return v
}

// DefaultOptions caches no data, reloads on updates, and uses MMAP.
var DefaultOptions = &Options{
	Caching:        CacheDefault,
	ReloadOnUpdate: true,
	UseMMap:        true,
}

// Record is a GeoIP record.
type Record struct {
	CountryCode   string  // CountryCode is a two-letter country code.
	CountryCode3  string  // CountryCode3 is a three-letter country code.
	CountryName   string  // CountryName is the name of the country.
	Region        string  // Region is the geographical region of the location.
	City          string  // City is the name of the city.
	PostalCode    string  // PostalCode is the location's postal code.
	Latitude      float64 // Latitude is the location's latitude.
	Longitude     float64 // Longitude is the location's longitude.
	AreaCode      int     // AreaCode is the location's area code.
	ContinentCode string  // ContinentCode is the location's continent.
}

// DB is a GeoIP database.
type DB struct {
	g *C.GeoIP
}

// Open returns an open DB instance of the given .dat file. The result *must* be
// closed, or memory will leak.
func Open(filename string, opts *Options) (*DB, error) {
	if opts == nil {
		opts = DefaultOptions
	}

	cs := C.CString(filename)
	defer C.free(unsafe.Pointer(cs))

	g, err := C.GeoIP_open(cs, C.int(opts.bitmask()))
	if err != nil {
		return nil, err
	}
	C.GeoIP_set_charset(g, C.GEOIP_CHARSET_UTF8)

	db := &DB{g: g}
	runtime.SetFinalizer(g, db.Close)
	return db, nil
}

// Lookup returns a GeoIP Record for the given IP address.
func (db *DB) Lookup(ip string) *Record {
	cs := C.CString(ip)
	defer C.free(unsafe.Pointer(cs))

	r := C.GeoIP_record_by_addr(db.g, cs)
	if r == nil {
		return nil
	}
	defer C.GeoIPRecord_delete(r)

	return &Record{
		CountryCode:   C.GoString(r.country_code),
		CountryCode3:  C.GoString(r.country_code3),
		CountryName:   C.GoString(r.country_name),
		Region:        C.GoString(r.region),
		City:          C.GoString(r.city),
		PostalCode:    C.GoString(r.postal_code),
		Latitude:      float64(r.latitude),
		Longitude:     float64(r.longitude),
		AreaCode:      int(r.area_code),
		ContinentCode: C.GoString(r.continent_code),
	}
}

// Close frees the memory associated with the DB.
func (db *DB) Close() error {
	if db.g != nil {
		C.GeoIP_delete(db.g)
	}
	db.g = nil
	return nil
}
