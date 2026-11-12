from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.connect_import_source_tone import ConnectImportSourceTone






T = TypeVar("T", bound="ConnectImportSource")



@_attrs_define
class ConnectImportSource:
    """ 
        Attributes:
            badge (str):
            detail (str):
            key (str):
            name (str):
            tone (ConnectImportSourceTone):
     """

    badge: str
    detail: str
    key: str
    name: str
    tone: ConnectImportSourceTone





    def to_dict(self) -> dict[str, Any]:
        badge = self.badge

        detail = self.detail

        key = self.key

        name = self.name

        tone = self.tone.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "badge": badge,
            "detail": detail,
            "key": key,
            "name": name,
            "tone": tone,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        badge = d.pop("badge")

        detail = d.pop("detail")

        key = d.pop("key")

        name = d.pop("name")

        tone = ConnectImportSourceTone(d.pop("tone"))




        connect_import_source = cls(
            badge=badge,
            detail=detail,
            key=key,
            name=name,
            tone=tone,
        )

        return connect_import_source

