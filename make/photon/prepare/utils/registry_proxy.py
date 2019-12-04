import os, copy

from g import config_dir, templates_dir, DEFAULT_GID, DEFAULT_UID
from utils.misc import prepare_dir
from utils.jinja import render_jinja


registry_config_dir = os.path.join(config_dir, "registry-proxy")
registry_config_template_path = os.path.join(templates_dir, "registry-proxy", "config.yml.jinja")
registry_conf = os.path.join(config_dir, "registry-proxy", "config.yml")


def prepare_registry_proxy(config_dict):
    prepare_dir(registry_config_dir)

    render_jinja(
        registry_config_template_path,
        registry_conf,
        uid=DEFAULT_UID,
        gid=DEFAULT_GID,
        **config_dict)

