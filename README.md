# winfastnav
A very WIP **fas**t-**nav**igation-bar for **Win**dows.

---

## About

The personal purpose of this project is to learn the Go language.

The purpose of the program is to act as a quick navigation bar, similarly to PowerToys Run.

## Screenshots

![imagen](https://github.com/user-attachments/assets/ac1276a1-d4e1-4454-8690-d120f99d7c50)

![imagen](https://github.com/user-attachments/assets/15fbcc5a-0844-4534-baff-3803c4678f79)

![imagen](https://github.com/user-attachments/assets/45dd0a16-b484-41b6-b4c5-014b4926bb13)

![imagen](https://github.com/user-attachments/assets/4b60aefa-50ee-471f-be7f-7d53937c007b)

## Build

go build -trimpath -ldflags="-H windowsgui -s -w" -o winfastnav.exe


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