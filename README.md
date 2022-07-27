# ctun1

netsh interface ip set address name="wintun" source=static addr=192.168.1.244 mask=255.255.255.0 gateway=none

route  add 0.0.0.0 mask 0.0.0.0 192.168.1.244 if 56 metric 5
route  add 192.168.74.128 mask 255.255.255.255 192.168.74.1