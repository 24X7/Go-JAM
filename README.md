# Go-JAM(Stack)
## Cloud Run Service Teamplate w/ Angular Universal & Golang Fiber Backplane

> This is a work in progress

Pretty simple proof of concept:
1. One exposed endpoint based on the `PORT` environment variable; other internal ports to the container are 4100 and 4000
2. Golang Fiber service is the front door to handle compression, service to service auth, storage (key value pairs)
3. Anuglar Universal App w/ Express Engine for application logic, user auth (setup via WorkOS service), and UI

Location build/dev server, and deployment (not done yet) handled through the file `./service`

Setup Development Environment:
1. Install golang (brew install golang)
2. Install NodeJS w/ NPM (brew install node)
3. Install `xz` as global (npm install -g xz)
4. Install levelDB form google for development (brew install leveldb)
5. chmod 777 ./service

Key commands:
1. `./service dev` (runs dev server)
2. `./service build` (performs production build including generating a macos binary for local testing)
3. `./service start-macos` (starts production build on mac from the build output in number 2 above)
4. `./service start` (starts the production linux instance to be used inside the docker container)

*Languages and Frameworks Used:* Golong, Go Fiber, Angular Universal w/ Express Engine, Google's `xz` nodejs package for automation and official node container for runtime.

*GCP Technologies:* Google Cloud Run, Google Secret Manager

What is left for initial POC?
1. Test building container
2. Setup Secret Manager
3. Switch the LevelDB store to use Google CLoud Storage
4. Setup the magic link endpoints using WorkOS
