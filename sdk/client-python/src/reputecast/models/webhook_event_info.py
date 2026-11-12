from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WebhookEventInfo")



@_attrs_define
class WebhookEventInfo:
    """ 
        Attributes:
            description (str):
            group (str):  Example: Email.
            name (str):  Example: email.bounced.
     """

    description: str
    group: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        group = self.group

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "group": group,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        group = d.pop("group")

        name = d.pop("name")

        webhook_event_info = cls(
            description=description,
            group=group,
            name=name,
        )

        return webhook_event_info

