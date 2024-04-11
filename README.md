# Mantium

Mantium is a dashboard for tracking from multiple sites, like [Mangadex](mangadex.org) and [ComicK](comick.io).

![image](https://github.com/diogovalentte/mantium/assets/49578155/69e5d417-e3c8-4a4e-9613-b47eff54ecce)

# Project structure and configuration

The project is divided in two: the **API** and the **dashboard**.

## API

After starting the API, you can find the API docs under the path `/v1/swagger/index.html`, like `http://192.168.1.44/v1/swagger/index.html` or `https://sub.domain.com/v1/swagger/index.html`, depending on how you access the API.

You can set the API to automatically update the mangas metadata (last upload chapter, cover image, etc.) periodically. Use environment variables to configure.

# Running

By default, the API will be available on port `8080` and the dashboard in port `8501`, and they're not accessible by other machines. To be accessible by other machines, you need to run the containers behind a reverse proxy or run the containers in [host network mode](https://docs.docker.com/network/drivers/host/).

- The API need to run on host only if you want to use the dashboard iFrame. More about the iFrame below.

- You can change the API and dashboard ports using the environment variables `API_PORT` and `DASHBOARD_PORT`.

## Docker Compose

1. There is a `docker-compose.yml` file in this repository. You can clone this repository to use this file or create one yourself.
1. Create a `.env` file, it should be like the `.env.example` file and be in the same directory as the `docker-compose.yml` file.
1. Start the containers:

```sh
docker compose up -d
```

## Manually

1. Export the environment variables in the `.env.example` file.

### API

2. Inside the `api/` folder, install the API dependencies:

```bash
go mod download
```

3. Start the API:

```bash
go run main.go
```

### Dashboard

4. Inside the `dashboard/` folder, install the dashboard dependencies:

```bash
pip install -r requirements.txt
```

5. Start the dashboard:

```bash
streamlit run 01_📖_Dashboard.py
```

6. Access the dashboard on `http://localhost:8501`

# IMPORTANT!

- The dashboard and the API don't have any authentication system, so anyone who can access the API will be able to do whatever they want. You can add an authentication portal like [Authelia](https://github.com/authelia/authelia) or [Authentik](https://github.com/goauthentik/authentik) in front of the dashboard to protect it and don't expose the API. If you want to use the iFrame returned by the API, you can also put an authentication portal in front of it if the API and dashboard containers are in the same Docker network. The dashboard will communicate with the API using the API's container name.

# Homarr iFrame

The API has an endpoint that returns an HTML code to be used as iFrame (designed to be used in [Homarr](https://github.com/ajnart/homarr)).

- **IMPORTANT**: check the API docs to see which query arguments the iFrame endpoint needs.

![image](https://github.com/diogovalentte/mantium/assets/49578155/188ab83a-57a4-400d-92c6-1f935728ef50)

When you add an iFrame widget in your Homarr dashboard, it's **>your<** web browser that fetches the HTML content from the API and shows it to you, not Homarr. So your browser needs to be able to access the API, that's how an iFrame works.

- **Examples**:
  - If you run the API on your server, you need to add your server IP address + port in the Homarr widget, and you need to make sure your browser can access this IP + port.
  - If you're accessing Homarr with a domain and using HTTPS, you also need to access this API with a domain and using HTTPS. If you try to use HTTP with your HTTPS, your browser will block the iFrame.
