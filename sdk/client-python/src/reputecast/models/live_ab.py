from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_ab_phase import LiveABPhase
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.live_ab_variant import LiveABVariant





T = TypeVar("T", bound="LiveAB")



@_attrs_define
class LiveAB:
    """ 
        Attributes:
            kind (str):
            leader (str):
            metric (str):
            phase (LiveABPhase):
            sample (int):
            variants (list[LiveABVariant]):
            winner (str):
            decide_at (datetime.datetime | Unset):
     """

    kind: str
    leader: str
    metric: str
    phase: LiveABPhase
    sample: int
    variants: list[LiveABVariant]
    winner: str
    decide_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_ab_variant import LiveABVariant # noqa: PLC0415
        kind = self.kind

        leader = self.leader

        metric = self.metric

        phase = self.phase.value

        sample = self.sample

        variants = []
        for variants_item_data in self.variants:
            variants_item = variants_item_data.to_dict()
            variants.append(variants_item)



        winner = self.winner

        decide_at: str | Unset = UNSET
        if not isinstance(self.decide_at, Unset):
            decide_at = self.decide_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
            "leader": leader,
            "metric": metric,
            "phase": phase,
            "sample": sample,
            "variants": variants,
            "winner": winner,
        })
        if decide_at is not UNSET:
            field_dict["decideAt"] = decide_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_ab_variant import LiveABVariant # noqa: PLC0415
        d = dict(src_dict)
        kind = d.pop("kind")

        leader = d.pop("leader")

        metric = d.pop("metric")

        phase = LiveABPhase(d.pop("phase"))




        sample = d.pop("sample")

        variants = []
        _variants = d.pop("variants")
        for variants_item_data in (_variants):
            variants_item = LiveABVariant.from_dict(variants_item_data)



            variants.append(variants_item)


        winner = d.pop("winner")

        _decide_at = d.pop("decideAt", UNSET)
        decide_at: datetime.datetime | Unset
        if isinstance(_decide_at,  Unset):
            decide_at = UNSET
        else:
            decide_at = datetime.datetime.fromisoformat(_decide_at)




        live_ab = cls(
            kind=kind,
            leader=leader,
            metric=metric,
            phase=phase,
            sample=sample,
            variants=variants,
            winner=winner,
            decide_at=decide_at,
        )

        return live_ab

