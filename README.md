# coronagraph

Coronagraph is an authenticating proxy whose sole purpose is to mitigate credential theft via malware (e.g. distributed via supply chain attacks) or other forms of local access (e.g. exploit). Its goal is to keep sensitive credentials entirely off disk and otherwise irretrievable. When configured correctly, malware that scrapes the filesystem for credentials will not have anything useful to steal.

It does this by decrypting secrets in DotEnv format and holding onto them in plaintext only in memory. When the proxy reaches a service Coronagraph knows about, it will transparently handle authentication by fetching credentials from this secrets store and inserting them into the outbound HTTP request. The client never touches a live credential; the request is authenticated by the time it arrives at the origin.

Additionally, for services Coronagraph recognizes, it can quietly authenticate read-only requests but escalate to the user via TouchID for read-write requests. For example, when configured with RubyGems, we can have package searches/fetches take place quietly (including for private repositories that require authentication), and prompt for TouchID if a gem is being pushed or yanked. This way, we not only mitigate credential theft, but also credential abuse in scenarios where a malicious payload attempts privileged actions (e.g. pushing malware back upstream, worm style)

## Setup (CA Certificates)

Because Coronagraph inserts credentials into HTTP requests, it needs to be able to terminate TLS. To do that, we'll use [mkcert](https://github.com/filosottile/mkcert). Generate your certs with it, and if you like, install them to your operating system trust store. Depending on what HTTP clients you use with Coronagraph, this may make your life a lot easier. Otherwise, you may need to point each client at your new certificates.

```
$ mkcert
$ mkcert -install
```

## Setup coronagraph

Coronagraph is a single Golang binary. Run `make` in the root directory to get started. Before you can use the proxy, you will need to set up credentials:

```
cg local-keys init
cg local-keys edit
```

Init will prompt you to set a passphrase, and all subsequent `local-keys` commands will expect the same one. Edit will pop up a vim editor where you can place your credentials in [Dot Env format](https://www.dotenv.org/docs/security/env.html). Right now, only a local passphrase backed vault (Argon2 for KDF, used to encrypt a randomly generated AES key in GCM mode that's rotated on every write) is supported.

To start the proxy, you will need to point it at the CA Certificate you made with `mkcert` and enter your passphrase from earlier.

```
cg serve --ca-cert ~/Library/Application\ Support/mkcert/rootCA.pem --ca-cert-key ~/Library/Application\ Support/mkcert/rootCA-key.pem
```

By default, it will listen on `localhost:11111`. At this point, you can point any HTTP client at this proxy. If you did not install the CA certificate and key to your system trust store, you will also need to configure each client to find them.

## Credential Support

Right now, coronagraph supports Rubygems (gem and bundler) and Github. To set them up, use `cg local-keys edit`.

* Rubygems: `GEM_HOST_API_KEY`
- `export GEM_HOST_API_KEY=placeholder`
- `gem yank -p http://localhost:1111 test-gem -v 1.2.3`

* Github: `GH_TOKEN`
- `export HTTP_PROXY=localhost:11111 HTTPS_PROXY=localhost:11111 GH_TOKEN=placeholder`
- `gh status`
