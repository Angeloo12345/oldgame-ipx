# oldgame-ipx

Náš vlastní **IPX relay** pro multiplayer hraní DOS her v prohlížeči na [oldgame.cz](https://www.oldgame.cz)
(js-dos / DOSBox IPX přes WebSocket). Přeposílá pakety mezi hráči ve stejné „místnosti".

Vychází z [caiiiycuk/dosbox-ipx-server](https://github.com/caiiiycuk/dosbox-ipx-server) (MIT),
upravené tak, aby si vzalo TCP port z `$PORT` (kvůli Renderu/PaaS, který sám řeší TLS).

## Endpoint
```
wss://<host>/ipx/<room>
```
Klient (play.php) volá `ci.networkConnect(0, "wss://<host>/ipx/" + room)`.

## Nasazení na Render.com (free, bez karty)
1. Přihlas se na https://render.com přes GitHub.
2. **New → Blueprint** → vyber tento repozitář → **Apply**.
   (Render přečte `render.yaml`, postaví Go binárku a spustí ji.)
3. Až naběhne, zkopíruj URL `https://oldgame-ipx-XXXX.onrender.com`.

> Free služba po ~15 min nečinnosti usne; probudí se sama při dalším spojení
> (pár vteřin). Aby neusínala, stačí venkovní ping každých ~10 min
> (cron-job.org / UptimeRobot) na `/`.

## Lokálně / na VPS
```
cd src && go build -o relay . && ./relay            # plain ws na :1900
./relay -c cert.pem -k key.pem                       # přímo wss na :1900
```
