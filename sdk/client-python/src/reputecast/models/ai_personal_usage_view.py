from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_personal_usage_view_state import AIPersonalUsageViewState
from typing import cast
import datetime






T = TypeVar("T", bound="AIPersonalUsageView")



@_attrs_define
class AIPersonalUsageView:
    """ 
        Attributes:
            cost_micros (int):
            requests (int):
            reserved_cost_micros (int):
            resets_at (datetime.datetime):
            share (float): Share of the cap used, 0 to 1
            state (AIPersonalUsageViewState):
            uncertain_requests (int):
     """

    cost_micros: int
    requests: int
    reserved_cost_micros: int
    resets_at: datetime.datetime
    share: float
    state: AIPersonalUsageViewState
    uncertain_requests: int





    def to_dict(self) -> dict[str, Any]:
        cost_micros = self.cost_micros

        requests = self.requests

        reserved_cost_micros = self.reserved_cost_micros

        resets_at = self.resets_at.isoformat()

        share = self.share

        state = self.state.value

        uncertain_requests = self.uncertain_requests


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "costMicros": cost_micros,
            "requests": requests,
            "reservedCostMicros": reserved_cost_micros,
            "resetsAt": resets_at,
            "share": share,
            "state": state,
            "uncertainRequests": uncertain_requests,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_micros = d.pop("costMicros")

        requests = d.pop("requests")

        reserved_cost_micros = d.pop("reservedCostMicros")

        resets_at = datetime.datetime.fromisoformat(d.pop("resetsAt"))




        share = d.pop("share")

        state = AIPersonalUsageViewState(d.pop("state"))




        uncertain_requests = d.pop("uncertainRequests")

        ai_personal_usage_view = cls(
            cost_micros=cost_micros,
            requests=requests,
            reserved_cost_micros=reserved_cost_micros,
            resets_at=resets_at,
            share=share,
            state=state,
            uncertain_requests=uncertain_requests,
        )

        return ai_personal_usage_view

