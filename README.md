# winfastnav
A very WIP **fas**t-**nav**igation-bar for **Win**dows.

---

## About

The personal purpose of this project is to learn the Go language.

The purpose of the program is to act as a quick navigation bar, similarly to PowerToys Run.

## TODO

- Navigating open windows
- Shortcuts to some functionality (ie. internet search)
- Settings and customization
- Blacklisting applications
- Auto-start with Windows
- Auto-update
- Look into lowering memory usage (Currently ~50mb which is quite brutal for such a simple application)

## Build

go build -ldflags="-H windowsgui -s -w" -o winfastnav.exe


## Never asked questions

- Q: Why learn Go?
- A: It seems like fun, and a good balance between a managed language and a systems language as far as ease-of-use to performance ratio goes.


- Q: Why did you do X thing Z way?
- A: Because I am learning and don't know better. If I've done something wrong or stupid, please let me know.


- Q: Feature?

- A: Suggest in issue or implement and PR


- Q: Multiplatform?

- A: The only platform-specific stuff is the application finder, so it'd be doable. However, MacOS and most Linux distributions already have a competent launcher bar of this style, so I don't see the point.


- Q: Licence?
- A: Refer to my [current general licence](https://markski.ar/general_licence.txt).