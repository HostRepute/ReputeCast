from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="WebhookInput")



@_attrs_define
class WebhookInput:
    """ 
        Attributes:
            events (list[str]):
            name (str):
            url (str):
            batch_email (bool | Unset):
            description (str | Unset):
     """

    events: list[str]
    name: str
    url: str
    batch_email: bool | Unset = UNSET
    description: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        events = self.events



        name = self.name

        url = self.url

        batch_email = self.batch_email

        description = self.description


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
            "name": name,
            "url": url,
        })
        if batch_email is not UNSET:
            field_dict["batchEmail"] = batch_email
        if description is not UNSET:
            field_dict["description"] = description

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        events = cast(list[str], d.pop("events"))


        name = d.pop("name")

        url = d.pop("url")

        batch_email = d.pop("batchEmail", UNSET)

        description = d.pop("description", UNSET)

        webhook_input = cls(
            events=events,
            name=name,
            url=url,
            batch_email=batch_email,
            description=description,
        )

        return webhook_input

