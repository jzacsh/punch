var server = require('./server.js');
var respond = require('./router.js').respond; //@TODO: ref this file in ./router/
var punch = require('./punch.js');
var options = require('./options.js').options;

server.start(respond, options);
console.log('Server initiated with config: %s', options);
