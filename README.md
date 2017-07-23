Setaria
=======

Seraria is a simple static blog engine that written in go-lang.


Install
-------

```
$ go get github.com/seamile/setaria
```


Usage
-----

Basic usage:
```
$ setaria
```

You can specify these params as you need:

```
-host   your host ip or hostname      (default: localhost)
-port   the server port for listening (default: 8000)
-notes  the blog files' storage path  (default: ./Setaria)
-theme  blog theme                    (default: simple)
```

Note Format
-----------

The format of note files is a subset of markdown.


TODO
----

- Auto reload when note files changed
- Search
- more themes
