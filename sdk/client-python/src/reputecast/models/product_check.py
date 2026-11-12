from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.product_check_key import ProductCheckKey
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductCheck")



@_attrs_define
class ProductCheck:
    """ 
        Attributes:
            key (ProductCheckKey):
            ok (bool):
            account (str | Unset):
            advisory (bool | Unset):
            at (datetime.datetime | Unset):
            bytes_ (int | Unset):
            default (bool | Unset): No sales page chosen: the default product page
            draft (bool | Unset): The template has a draft but no published version
            file (str | Unset):
            mode (str | Unset):
            path (str | Unset):
            route (str | Unset):  Example: Postmark, then Amazon SES.
            template (str | Unset):
            template_id (str | Unset):
            version (int | Unset):
     """

    key: ProductCheckKey
    ok: bool
    account: str | Unset = UNSET
    advisory: bool | Unset = UNSET
    at: datetime.datetime | Unset = UNSET
    bytes_: int | Unset = UNSET
    default: bool | Unset = UNSET
    draft: bool | Unset = UNSET
    file: str | Unset = UNSET
    mode: str | Unset = UNSET
    path: str | Unset = UNSET
    route: str | Unset = UNSET
    template: str | Unset = UNSET
    template_id: str | Unset = UNSET
    version: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key.value

        ok = self.ok

        account = self.account

        advisory = self.advisory

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        bytes_ = self.bytes_

        default = self.default

        draft = self.draft

        file = self.file

        mode = self.mode

        path = self.path

        route = self.route

        template = self.template

        template_id = self.template_id

        version = self.version


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "ok": ok,
        })
        if account is not UNSET:
            field_dict["account"] = account
        if advisory is not UNSET:
            field_dict["advisory"] = advisory
        if at is not UNSET:
            field_dict["at"] = at
        if bytes_ is not UNSET:
            field_dict["bytes"] = bytes_
        if default is not UNSET:
            field_dict["default"] = default
        if draft is not UNSET:
            field_dict["draft"] = draft
        if file is not UNSET:
            field_dict["file"] = file
        if mode is not UNSET:
            field_dict["mode"] = mode
        if path is not UNSET:
            field_dict["path"] = path
        if route is not UNSET:
            field_dict["route"] = route
        if template is not UNSET:
            field_dict["template"] = template
        if template_id is not UNSET:
            field_dict["templateId"] = template_id
        if version is not UNSET:
            field_dict["version"] = version

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = ProductCheckKey(d.pop("key"))




        ok = d.pop("ok")

        account = d.pop("account", UNSET)

        advisory = d.pop("advisory", UNSET)

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        bytes_ = d.pop("bytes", UNSET)

        default = d.pop("default", UNSET)

        draft = d.pop("draft", UNSET)

        file = d.pop("file", UNSET)

        mode = d.pop("mode", UNSET)

        path = d.pop("path", UNSET)

        route = d.pop("route", UNSET)

        template = d.pop("template", UNSET)

        template_id = d.pop("templateId", UNSET)

        version = d.pop("version", UNSET)

        product_check = cls(
            key=key,
            ok=ok,
            account=account,
            advisory=advisory,
            at=at,
            bytes_=bytes_,
            default=default,
            draft=draft,
            file=file,
            mode=mode,
            path=path,
            route=route,
            template=template,
            template_id=template_id,
            version=version,
        )

        return product_check

