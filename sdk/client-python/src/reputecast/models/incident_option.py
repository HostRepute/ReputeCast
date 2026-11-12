from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="IncidentOption")



@_attrs_define
class IncidentOption:
    """ 
        Attributes:
            action (str):
            body (str):
            key (str):
            recommended (bool):
            title (str):
     """

    action: str
    body: str
    key: str
    recommended: bool
    title: str





    def to_dict(self) -> dict[str, Any]:
        action = self.action

        body = self.body

        key = self.key

        recommended = self.recommended

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "body": body,
            "key": key,
            "recommended": recommended,
            "title": title,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = d.pop("action")

        body = d.pop("body")

        key = d.pop("key")

        recommended = d.pop("recommended")

        title = d.pop("title")

        incident_option = cls(
            action=action,
            body=body,
            key=key,
            recommended=recommended,
            title=title,
        )

        return incident_option

