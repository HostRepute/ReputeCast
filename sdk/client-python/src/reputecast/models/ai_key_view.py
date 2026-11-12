from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AIKeyView")



@_attrs_define
class AIKeyView:
    """ 
        Attributes:
            set_ (bool):
            added_at (datetime.datetime | Unset):
            added_by (str | Unset):
     """

    set_: bool
    added_at: datetime.datetime | Unset = UNSET
    added_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        set_ = self.set_

        added_at: str | Unset = UNSET
        if not isinstance(self.added_at, Unset):
            added_at = self.added_at.isoformat()

        added_by = self.added_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "set": set_,
        })
        if added_at is not UNSET:
            field_dict["addedAt"] = added_at
        if added_by is not UNSET:
            field_dict["addedBy"] = added_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        set_ = d.pop("set")

        _added_at = d.pop("addedAt", UNSET)
        added_at: datetime.datetime | Unset
        if isinstance(_added_at,  Unset):
            added_at = UNSET
        else:
            added_at = datetime.datetime.fromisoformat(_added_at)




        added_by = d.pop("addedBy", UNSET)

        ai_key_view = cls(
            set_=set_,
            added_at=added_at,
            added_by=added_by,
        )

        return ai_key_view

