from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="AutomationWebhook")



@_attrs_define
class AutomationWebhook:
    """ 
        Attributes:
            demo (bool | Unset):
            has_secret (bool | Unset):
            method (str | Unset):
            name (str | Unset):
            url (str | Unset):
     """

    demo: bool | Unset = UNSET
    has_secret: bool | Unset = UNSET
    method: str | Unset = UNSET
    name: str | Unset = UNSET
    url: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        demo = self.demo

        has_secret = self.has_secret

        method = self.method

        name = self.name

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if demo is not UNSET:
            field_dict["demo"] = demo
        if has_secret is not UNSET:
            field_dict["hasSecret"] = has_secret
        if method is not UNSET:
            field_dict["method"] = method
        if name is not UNSET:
            field_dict["name"] = name
        if url is not UNSET:
            field_dict["url"] = url

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        demo = d.pop("demo", UNSET)

        has_secret = d.pop("hasSecret", UNSET)

        method = d.pop("method", UNSET)

        name = d.pop("name", UNSET)

        url = d.pop("url", UNSET)

        automation_webhook = cls(
            demo=demo,
            has_secret=has_secret,
            method=method,
            name=name,
            url=url,
        )

        return automation_webhook

