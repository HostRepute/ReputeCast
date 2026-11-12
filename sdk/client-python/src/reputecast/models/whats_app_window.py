from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppWindow")



@_attrs_define
class WhatsAppWindow:
    """ 
        Attributes:
            open_ (bool):
            last_at (datetime.datetime | Unset):
            last_text (str | Unset):
            until (datetime.datetime | Unset):
     """

    open_: bool
    last_at: datetime.datetime | Unset = UNSET
    last_text: str | Unset = UNSET
    until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        open_ = self.open_

        last_at: str | Unset = UNSET
        if not isinstance(self.last_at, Unset):
            last_at = self.last_at.isoformat()

        last_text = self.last_text

        until: str | Unset = UNSET
        if not isinstance(self.until, Unset):
            until = self.until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "open": open_,
        })
        if last_at is not UNSET:
            field_dict["lastAt"] = last_at
        if last_text is not UNSET:
            field_dict["lastText"] = last_text
        if until is not UNSET:
            field_dict["until"] = until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        open_ = d.pop("open")

        _last_at = d.pop("lastAt", UNSET)
        last_at: datetime.datetime | Unset
        if isinstance(_last_at,  Unset):
            last_at = UNSET
        else:
            last_at = datetime.datetime.fromisoformat(_last_at)




        last_text = d.pop("lastText", UNSET)

        _until = d.pop("until", UNSET)
        until: datetime.datetime | Unset
        if isinstance(_until,  Unset):
            until = UNSET
        else:
            until = datetime.datetime.fromisoformat(_until)




        whats_app_window = cls(
            open_=open_,
            last_at=last_at,
            last_text=last_text,
            until=until,
        )

        return whats_app_window

