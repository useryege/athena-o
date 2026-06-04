# Documentation Site

## Developing And Testing

The [documentation website](https://athena.readthedocs.io/) is built using `mkdocs` and `mkdocs-material`.

To test:

```bash
make serve-docs-local
```
Once running, you can view your locally built documentation at [http://127.0.0.1:8000/](http://127.0.0.1:8000/).
Making changes to documentation will automatically rebuild and refresh the view.

Before submitting a PR build the website, to verify that there are no errors building the site
```bash
make build-docs
```

If you want to build and test the site directly on your local machine, follow the below steps:

1. Install the dependencies from the root of this repository using the `pip` command
    ```bash
    pip install -r docs/requirements.txt
    ```
2. Build the docs site locally from the root
   ```bash
   mkdocs build
   ``` 
3. Start the docs site locally
   ```bash
   make serve-docs-local
   ```

## Analytics

> [!TIP]
> Don't forget to disable your ad-blocker when testing.

We collect [Google Analytics](https://analytics.google.com/analytics/web/#/report-home/a105170809w198079555p192782995).
