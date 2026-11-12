from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_usage_view_state import AIUsageViewState
from typing import cast
import datetime






T = TypeVar("T", bound="AIUsageView")



@_attrs_define
class AIUsageView:
    """ 
        Attributes:
            cost_micros (int):
            input_tokens (int):
            month_start (datetime.datetime):
            output_tokens (int):
            requests (int):
            resets_at (datetime.datetime):
            share (float): Share of the cap used, 0 to 1 (0 without a cap)
            state (AIUsageViewState):
            tokens (int):
     """

    cost_micros: int
    input_tokens: int
    month_start: datetime.datetime
    output_tokens: int
    requests: int
    resets_at: datetime.datetime
    share: float
    state: AIUsageViewState
    tokens: int





    def to_dict(self) -> dict[str, Any]:
        cost_micros = self.cost_micros

        input_tokens = self.input_tokens

        month_start = self.month_start.isoformat()

        output_tokens = self.output_tokens

        requests = self.requests

        resets_at = self.resets_at.isoformat()

        share = self.share

        state = self.state.value

        tokens = self.tokens


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "costMicros": cost_micros,
            "inputTokens": input_tokens,
            "monthStart": month_start,
            "outputTokens": output_tokens,
            "requests": requests,
            "resetsAt": resets_at,
            "share": share,
            "state": state,
            "tokens": tokens,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cost_micros = d.pop("costMicros")

        input_tokens = d.pop("inputTokens")

        month_start = datetime.datetime.fromisoformat(d.pop("monthStart"))




        output_tokens = d.pop("outputTokens")

        requests = d.pop("requests")

        resets_at = datetime.datetime.fromisoformat(d.pop("resetsAt"))




        share = d.pop("share")

        state = AIUsageViewState(d.pop("state"))




        tokens = d.pop("tokens")

        ai_usage_view = cls(
            cost_micros=cost_micros,
            input_tokens=input_tokens,
            month_start=month_start,
            output_tokens=output_tokens,
            requests=requests,
            resets_at=resets_at,
            share=share,
            state=state,
            tokens=tokens,
        )

        return ai_usage_view

