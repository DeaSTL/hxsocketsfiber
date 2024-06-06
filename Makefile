
build:
	mkdir -p ./bin
	npx tailwindcss --input ./static/css/tailwind.css --output ./static/css/tailwind-out.css
	templ generate ./views
	go build -o ./bin/main ./example/

run:
	./bin/main
