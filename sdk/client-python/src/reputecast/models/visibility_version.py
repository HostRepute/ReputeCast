from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="VisibilityVersion")



@_attrs_define
class VisibilityVersion:
    """ 
        Attributes:
            checked (bool):
            empty (bool):
            hidden (list[str]):
            number (int):
            over_clip (bool):
            recipients (int):
            shown (list[str]):
            shows (str):
            size_bytes (int):
            largest (bool | Unset):
            smallest (bool | Unset):
     """

    checked: bool
    empty: bool
    hidden: list[str]
    number: int
    over_clip: bool
    recipients: int
    shown: list[str]
    shows: str
    size_bytes: int
    largest: bool | Unset = UNSET
    smallest: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        checked = self.checked

        empty = self.empty

        hidden = self.hidden



        number = self.number

        over_clip = self.over_clip

        recipients = self.recipients

        shown = self.shown



        shows = self.shows

        size_bytes = self.size_bytes

        largest = self.largest

        smallest = self.smallest


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "checked": checked,
            "empty": empty,
            "hidden": hidden,
            "number": number,
            "overClip": over_clip,
            "recipients": recipients,
            "shown": shown,
            "shows": shows,
            "sizeBytes": size_bytes,
        })
        if largest is not UNSET:
            field_dict["largest"] = largest
        if smallest is not UNSET:
            field_dict["smallest"] = smallest

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        checked = d.pop("checked")

        empty = d.pop("empty")

        hidden = cast(list[str], d.pop("hidden"))


        number = d.pop("number")

        over_clip = d.pop("overClip")

        recipients = d.pop("recipients")

        shown = cast(list[str], d.pop("shown"))


        shows = d.pop("shows")

        size_bytes = d.pop("sizeBytes")

        largest = d.pop("largest", UNSET)

        smallest = d.pop("smallest", UNSET)

        visibility_version = cls(
            checked=checked,
            empty=empty,
            hidden=hidden,
            number=number,
            over_clip=over_clip,
            recipients=recipients,
            shown=shown,
            shows=shows,
            size_bytes=size_bytes,
            largest=largest,
            smallest=smallest,
        )

        return visibility_version

