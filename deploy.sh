scp main.go ubuntu@mpcabete.xyz:/home/ubuntu/mkk/main.go
ssh ubuntu@mpcabete.xyz "cd mkk && docker compose build --no-cache && docker compose up -d"
