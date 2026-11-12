from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.warmup_candidate import WarmupCandidate
  from ..models.warmup_plan_view import WarmupPlanView





T = TypeVar("T", bound="WarmupOverview")



@_attrs_define
class WarmupOverview:
    """ 
        Attributes:
            candidates (list[WarmupCandidate]):
            plans (list[WarmupPlanView]):
     """

    candidates: list[WarmupCandidate]
    plans: list[WarmupPlanView]





    def to_dict(self) -> dict[str, Any]:
        from ..models.warmup_candidate import WarmupCandidate # noqa: PLC0415
        from ..models.warmup_plan_view import WarmupPlanView # noqa: PLC0415
        candidates = []
        for candidates_item_data in self.candidates:
            candidates_item = candidates_item_data.to_dict()
            candidates.append(candidates_item)



        plans = []
        for plans_item_data in self.plans:
            plans_item = plans_item_data.to_dict()
            plans.append(plans_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "candidates": candidates,
            "plans": plans,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.warmup_candidate import WarmupCandidate # noqa: PLC0415
        from ..models.warmup_plan_view import WarmupPlanView # noqa: PLC0415
        d = dict(src_dict)
        candidates = []
        _candidates = d.pop("candidates")
        for candidates_item_data in (_candidates):
            candidates_item = WarmupCandidate.from_dict(candidates_item_data)



            candidates.append(candidates_item)


        plans = []
        _plans = d.pop("plans")
        for plans_item_data in (_plans):
            plans_item = WarmupPlanView.from_dict(plans_item_data)



            plans.append(plans_item)


        warmup_overview = cls(
            candidates=candidates,
            plans=plans,
        )

        return warmup_overview

