from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
from uuid import UUID
import datetime






T = TypeVar("T", bound="CountdownImage")



@_attrs_define
class CountdownImage:
    """ 
        Attributes:
            alt (str):  Example: Offer ends Fri 28 Nov at 23:59 (EST).
            ends_at (datetime.datetime):
            height (int):
            id (UUID):
            url (str): The animated GIF, drawn when the email is opened
            width (int):
     """

    alt: str
    ends_at: datetime.datetime
    height: int
    id: UUID
    url: str
    width: int





    def to_dict(self) -> dict[str, Any]:
        alt = self.alt

        ends_at = self.ends_at.isoformat()

        height = self.height

        id = str(self.id)

        url = self.url

        width = self.width


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "alt": alt,
            "endsAt": ends_at,
            "height": height,
            "id": id,
            "url": url,
            "width": width,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        alt = d.pop("alt")

        ends_at = datetime.datetime.fromisoformat(d.pop("endsAt"))




        height = d.pop("height")

        id = UUID(d.pop("id"))




        url = d.pop("url")

        width = d.pop("width")

        countdown_image = cls(
            alt=alt,
            ends_at=ends_at,
            height=height,
            id=id,
            url=url,
            width=width,
        )

        return countdown_image

