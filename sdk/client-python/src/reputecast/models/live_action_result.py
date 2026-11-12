from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.live_snapshot import LiveSnapshot





T = TypeVar("T", bound="LiveActionResult")



@_attrs_define
class LiveActionResult:
    """ 
        Attributes:
            cancelled (int):
            decided (int):
            excluded (int):
            moved (int):
            snapshot (LiveSnapshot):
     """

    cancelled: int
    decided: int
    excluded: int
    moved: int
    snapshot: LiveSnapshot





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_snapshot import LiveSnapshot # noqa: PLC0415
        cancelled = self.cancelled

        decided = self.decided

        excluded = self.excluded

        moved = self.moved

        snapshot = self.snapshot.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cancelled": cancelled,
            "decided": decided,
            "excluded": excluded,
            "moved": moved,
            "snapshot": snapshot,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_snapshot import LiveSnapshot # noqa: PLC0415
        d = dict(src_dict)
        cancelled = d.pop("cancelled")

        decided = d.pop("decided")

        excluded = d.pop("excluded")

        moved = d.pop("moved")

        snapshot = LiveSnapshot.from_dict(d.pop("snapshot"))




        live_action_result = cls(
            cancelled=cancelled,
            decided=decided,
            excluded=excluded,
            moved=moved,
            snapshot=snapshot,
        )

        return live_action_result

