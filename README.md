# Go old monolith

A collection of services from my distant past rewritten in go
and all stuck together in one monolithic service.

I simply don't care much about these and don't want to individually
manage them as we inexorably move servers throughout the future.

## About

This is a monolithic collection of services, but each service is 
isolated from one another such that they COULD be split out if 
desired. Each service is a separate package that exposes some
function to generate the http handler which can be mounted to
some endpoint at the top level. 

All services use the same config file, `config.toml`, for ease 
of management. However, each service doesn't know it gets its 
config from the same file, as each is configured with a unique
and independent struct.

All files are set to write next to the service, which may not
be desired in all instances. It's desired here: it's easier to
move the entire thing when everything is in the same place.

All services should enable limits to prevent misuse of the 
filesystem. These are old services nobody is using anymore
(except maybe me sometimes), so limits should be highly
restrictive. For instance, the `webstream` service only allows
a limited number of rooms to be created on the filesystem, 
and only allows a very small number of active rooms in memory.
Many of these services were written _without_ limits in mind,
so some larger modifications are sometimes necessary. These
may break some systems; if anybody somehow ends up using 
one of these, just create an issue and I'll fix it.

## Publishing

On the current production server (2024), there is a publish script
you can use. It should do everything other than actually creating
the service or setting up the nginx proxies.

### Caveats

Make sure you set appropriate nginx `proxy_read_timeout` values
for endpoints that have longpolling (you want the services to
time themselves out rather than nginx doing it).

If you have previously deployed the service and made updates to
the config, you will have to add the new values manually, or 
delete the config and regenerate it on startup.


