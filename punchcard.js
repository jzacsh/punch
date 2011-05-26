var server = require('./server.js');
var respond = require('./router.js').respond; //@TODO: ref this file in ./router/
var punch = require('./punch.js');
var config = require('./config.js').config;

server.start(respond, config);
console.log('Server initiated with config: %s', config);
