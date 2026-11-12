from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.synced_uses import SyncedUses





T = TypeVar("T", bound="SyncedDeleted")



@_attrs_define
class SyncedDeleted:
    """ 
        Attributes:
            applied (SyncedUses):
     """

    applied: SyncedUses





    def to_dict(self) -> dict[str, Any]:
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        applied = self.applied.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applied": applied,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        d = dict(src_dict)
        applied = SyncedUses.from_dict(d.pop("applied"))




        synced_deleted = cls(
            applied=applied,
        )

        return synced_deleted

