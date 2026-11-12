from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.synced_block import SyncedBlock
  from ..models.synced_uses import SyncedUses





T = TypeVar("T", bound="SyncedSaved")



@_attrs_define
class SyncedSaved:
    """ 
        Attributes:
            applied (SyncedUses):
            block (SyncedBlock):
     """

    applied: SyncedUses
    block: SyncedBlock





    def to_dict(self) -> dict[str, Any]:
        from ..models.synced_block import SyncedBlock # noqa: PLC0415
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        applied = self.applied.to_dict()

        block = self.block.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "applied": applied,
            "block": block,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.synced_block import SyncedBlock # noqa: PLC0415
        from ..models.synced_uses import SyncedUses # noqa: PLC0415
        d = dict(src_dict)
        applied = SyncedUses.from_dict(d.pop("applied"))




        block = SyncedBlock.from_dict(d.pop("block"))




        synced_saved = cls(
            applied=applied,
            block=block,
        )

        return synced_saved

