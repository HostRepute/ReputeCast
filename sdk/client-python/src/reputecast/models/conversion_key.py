from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ConversionKey")



@_attrs_define
class ConversionKey:
    """ 
        Attributes:
            created_at (datetime.datetime):
            created_by (str):
            id (str):
            name (str):
            only (bool):
            prefix (str):
            expires_at (datetime.datetime | Unset):
            last_used_at (datetime.datetime | Unset):
            last_used_ip (str | Unset):
     """

    created_at: datetime.datetime
    created_by: str
    id: str
    name: str
    only: bool
    prefix: str
    expires_at: datetime.datetime | Unset = UNSET
    last_used_at: datetime.datetime | Unset = UNSET
    last_used_ip: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        created_at = self.created_at.isoformat()

        created_by = self.created_by

        id = self.id

        name = self.name

        only = self.only

        prefix = self.prefix

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        last_used_at: str | Unset = UNSET
        if not isinstance(self.last_used_at, Unset):
            last_used_at = self.last_used_at.isoformat()

        last_used_ip = self.last_used_ip


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "createdAt": created_at,
            "createdBy": created_by,
            "id": id,
            "name": name,
            "only": only,
            "prefix": prefix,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if last_used_at is not UNSET:
            field_dict["lastUsedAt"] = last_used_at
        if last_used_ip is not UNSET:
            field_dict["lastUsedIp"] = last_used_ip

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        id = d.pop("id")

        name = d.pop("name")

        only = d.pop("only")

        prefix = d.pop("prefix")

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _last_used_at = d.pop("lastUsedAt", UNSET)
        last_used_at: datetime.datetime | Unset
        if isinstance(_last_used_at,  Unset):
            last_used_at = UNSET
        else:
            last_used_at = datetime.datetime.fromisoformat(_last_used_at)




        last_used_ip = d.pop("lastUsedIp", UNSET)

        conversion_key = cls(
            created_at=created_at,
            created_by=created_by,
            id=id,
            name=name,
            only=only,
            prefix=prefix,
            expires_at=expires_at,
            last_used_at=last_used_at,
            last_used_ip=last_used_ip,
        )

        return conversion_key

