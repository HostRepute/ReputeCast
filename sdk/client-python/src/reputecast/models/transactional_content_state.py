from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TransactionalContentState")



@_attrs_define
class TransactionalContentState:
    """ 
        Attributes:
            available (bool):
            can_view (bool):
            helpers (list[str]):
            masked (list[str]):
            metadata_until (datetime.datetime):
            retention_days (int):
            expires_at (datetime.datetime | Unset):
            removed_at (datetime.datetime | Unset):
     """

    available: bool
    can_view: bool
    helpers: list[str]
    masked: list[str]
    metadata_until: datetime.datetime
    retention_days: int
    expires_at: datetime.datetime | Unset = UNSET
    removed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        available = self.available

        can_view = self.can_view

        helpers = self.helpers



        masked = self.masked



        metadata_until = self.metadata_until.isoformat()

        retention_days = self.retention_days

        expires_at: str | Unset = UNSET
        if not isinstance(self.expires_at, Unset):
            expires_at = self.expires_at.isoformat()

        removed_at: str | Unset = UNSET
        if not isinstance(self.removed_at, Unset):
            removed_at = self.removed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "available": available,
            "canView": can_view,
            "helpers": helpers,
            "masked": masked,
            "metadataUntil": metadata_until,
            "retentionDays": retention_days,
        })
        if expires_at is not UNSET:
            field_dict["expiresAt"] = expires_at
        if removed_at is not UNSET:
            field_dict["removedAt"] = removed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        available = d.pop("available")

        can_view = d.pop("canView")

        helpers = cast(list[str], d.pop("helpers"))


        masked = cast(list[str], d.pop("masked"))


        metadata_until = datetime.datetime.fromisoformat(d.pop("metadataUntil"))




        retention_days = d.pop("retentionDays")

        _expires_at = d.pop("expiresAt", UNSET)
        expires_at: datetime.datetime | Unset
        if isinstance(_expires_at,  Unset):
            expires_at = UNSET
        else:
            expires_at = datetime.datetime.fromisoformat(_expires_at)




        _removed_at = d.pop("removedAt", UNSET)
        removed_at: datetime.datetime | Unset
        if isinstance(_removed_at,  Unset):
            removed_at = UNSET
        else:
            removed_at = datetime.datetime.fromisoformat(_removed_at)




        transactional_content_state = cls(
            available=available,
            can_view=can_view,
            helpers=helpers,
            masked=masked,
            metadata_until=metadata_until,
            retention_days=retention_days,
            expires_at=expires_at,
            removed_at=removed_at,
        )

        return transactional_content_state

