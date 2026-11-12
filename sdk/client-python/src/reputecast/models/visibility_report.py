from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.visibility_block_info import VisibilityBlockInfo
  from ..models.visibility_version import VisibilityVersion





T = TypeVar("T", bound="VisibilityReport")



@_attrs_define
class VisibilityReport:
    """ 
        Attributes:
            blocks (list[VisibilityBlockInfo]):
            checked (int):
            counted (bool):
            duration_ms (int):
            invalid (list[str]):
            max_size (int):
            min_size (int):
            missing (int):
            possible (int):
            total (int):
            versions (list[VisibilityVersion]):
            counted_at (datetime.datetime | Unset):
     """

    blocks: list[VisibilityBlockInfo]
    checked: int
    counted: bool
    duration_ms: int
    invalid: list[str]
    max_size: int
    min_size: int
    missing: int
    possible: int
    total: int
    versions: list[VisibilityVersion]
    counted_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.visibility_block_info import VisibilityBlockInfo # noqa: PLC0415
        from ..models.visibility_version import VisibilityVersion # noqa: PLC0415
        blocks = []
        for blocks_item_data in self.blocks:
            blocks_item = blocks_item_data.to_dict()
            blocks.append(blocks_item)



        checked = self.checked

        counted = self.counted

        duration_ms = self.duration_ms

        invalid = self.invalid



        max_size = self.max_size

        min_size = self.min_size

        missing = self.missing

        possible = self.possible

        total = self.total

        versions = []
        for versions_item_data in self.versions:
            versions_item = versions_item_data.to_dict()
            versions.append(versions_item)



        counted_at: str | Unset = UNSET
        if not isinstance(self.counted_at, Unset):
            counted_at = self.counted_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "blocks": blocks,
            "checked": checked,
            "counted": counted,
            "durationMs": duration_ms,
            "invalid": invalid,
            "maxSize": max_size,
            "minSize": min_size,
            "missing": missing,
            "possible": possible,
            "total": total,
            "versions": versions,
        })
        if counted_at is not UNSET:
            field_dict["countedAt"] = counted_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.visibility_block_info import VisibilityBlockInfo # noqa: PLC0415
        from ..models.visibility_version import VisibilityVersion # noqa: PLC0415
        d = dict(src_dict)
        blocks = []
        _blocks = d.pop("blocks")
        for blocks_item_data in (_blocks):
            blocks_item = VisibilityBlockInfo.from_dict(blocks_item_data)



            blocks.append(blocks_item)


        checked = d.pop("checked")

        counted = d.pop("counted")

        duration_ms = d.pop("durationMs")

        invalid = cast(list[str], d.pop("invalid"))


        max_size = d.pop("maxSize")

        min_size = d.pop("minSize")

        missing = d.pop("missing")

        possible = d.pop("possible")

        total = d.pop("total")

        versions = []
        _versions = d.pop("versions")
        for versions_item_data in (_versions):
            versions_item = VisibilityVersion.from_dict(versions_item_data)



            versions.append(versions_item)


        _counted_at = d.pop("countedAt", UNSET)
        counted_at: datetime.datetime | Unset
        if isinstance(_counted_at,  Unset):
            counted_at = UNSET
        else:
            counted_at = datetime.datetime.fromisoformat(_counted_at)




        visibility_report = cls(
            blocks=blocks,
            checked=checked,
            counted=counted,
            duration_ms=duration_ms,
            invalid=invalid,
            max_size=max_size,
            min_size=min_size,
            missing=missing,
            possible=possible,
            total=total,
            versions=versions,
            counted_at=counted_at,
        )

        return visibility_report

