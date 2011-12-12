#punch

Time tracker named after old punch card time-clocks.

##Status
Abandoned/vaporware, as I've no time for this. So the node.js/socket.io bits will
probably never be written. Only the bash script exists at the moment.

##Background
Punch is based on a previously writen (now in the ```bin/``` dir) fully funcional
bash4 script called punch with an sqlite3 backend.

##Intention
This repo is just to play with *node.js*, *socket.io* and bash's *file descriptor*
interface (see ```man 1 bash | LESS +/REDIRECTION```). I'm thinking of IPC between
the cli (bash script) and the web interface (node.js) to allow for multiple
interfaces for one user.
