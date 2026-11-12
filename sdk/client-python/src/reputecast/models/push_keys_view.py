from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="PushKeysView")



@_attrs_define
class PushKeysView:
    """ 
        Attributes:
            contact (str):
            created_at (datetime.datetime):
            created_by (str):
            public_key (str):
            last_accepted_at (datetime.datetime | Unset):
     """

    contact: str
    created_at: datetime.datetime
    created_by: str
    public_key: str
    last_accepted_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact = self.contact

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        public_key = self.public_key

        last_accepted_at: str | Unset = UNSET
        if not isinstance(self.last_accepted_at, Unset):
            last_accepted_at = self.last_accepted_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contact": contact,
            "createdAt": created_at,
            "createdBy": created_by,
            "publicKey": public_key,
        })
        if last_accepted_at is not UNSET:
            field_dict["lastAcceptedAt"] = last_accepted_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact = d.pop("contact")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        public_key = d.pop("publicKey")

        _last_accepted_at = d.pop("lastAcceptedAt", UNSET)
        last_accepted_at: datetime.datetime | Unset
        if isinstance(_last_accepted_at,  Unset):
            last_accepted_at = UNSET
        else:
            last_accepted_at = datetime.datetime.fromisoformat(_last_accepted_at)




        push_keys_view = cls(
            contact=contact,
            created_at=created_at,
            created_by=created_by,
            public_key=public_key,
            last_accepted_at=last_accepted_at,
        )

        return push_keys_view

