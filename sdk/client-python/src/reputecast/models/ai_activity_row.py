from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.ai_activity_row_feature import AIActivityRowFeature
from ..models.ai_activity_row_outcome import AIActivityRowOutcome
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="AIActivityRow")



@_attrs_define
class AIActivityRow:
    """ 
        Attributes:
            at (datetime.datetime):
            cost_micros (int):
            feature (AIActivityRowFeature):
            id (str):
            input_tokens (int):
            member (str):
            model (str):
            on (str): What it was about: Weekly Digest #143
            outcome (AIActivityRowOutcome):
            output_tokens (int):
            provider (str):
            tokens (int):
            action (str | Unset): The rewrite's action, or the language translated to
            detail (str | Unset): 1 of 5, 529 overloaded
            member_color (str | Unset):
            target_id (str | Unset):
            target_kind (str | Unset):
            user_id (str | Unset):
     """

    at: datetime.datetime
    cost_micros: int
    feature: AIActivityRowFeature
    id: str
    input_tokens: int
    member: str
    model: str
    on: str
    outcome: AIActivityRowOutcome
    output_tokens: int
    provider: str
    tokens: int
    action: str | Unset = UNSET
    detail: str | Unset = UNSET
    member_color: str | Unset = UNSET
    target_id: str | Unset = UNSET
    target_kind: str | Unset = UNSET
    user_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        cost_micros = self.cost_micros

        feature = self.feature.value

        id = self.id

        input_tokens = self.input_tokens

        member = self.member

        model = self.model

        on = self.on

        outcome = self.outcome.value

        output_tokens = self.output_tokens

        provider = self.provider

        tokens = self.tokens

        action = self.action

        detail = self.detail

        member_color = self.member_color

        target_id = self.target_id

        target_kind = self.target_kind

        user_id = self.user_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "costMicros": cost_micros,
            "feature": feature,
            "id": id,
            "inputTokens": input_tokens,
            "member": member,
            "model": model,
            "on": on,
            "outcome": outcome,
            "outputTokens": output_tokens,
            "provider": provider,
            "tokens": tokens,
        })
        if action is not UNSET:
            field_dict["action"] = action
        if detail is not UNSET:
            field_dict["detail"] = detail
        if member_color is not UNSET:
            field_dict["memberColor"] = member_color
        if target_id is not UNSET:
            field_dict["targetId"] = target_id
        if target_kind is not UNSET:
            field_dict["targetKind"] = target_kind
        if user_id is not UNSET:
            field_dict["userId"] = user_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        cost_micros = d.pop("costMicros")

        feature = AIActivityRowFeature(d.pop("feature"))




        id = d.pop("id")

        input_tokens = d.pop("inputTokens")

        member = d.pop("member")

        model = d.pop("model")

        on = d.pop("on")

        outcome = AIActivityRowOutcome(d.pop("outcome"))




        output_tokens = d.pop("outputTokens")

        provider = d.pop("provider")

        tokens = d.pop("tokens")

        action = d.pop("action", UNSET)

        detail = d.pop("detail", UNSET)

        member_color = d.pop("memberColor", UNSET)

        target_id = d.pop("targetId", UNSET)

        target_kind = d.pop("targetKind", UNSET)

        user_id = d.pop("userId", UNSET)

        ai_activity_row = cls(
            at=at,
            cost_micros=cost_micros,
            feature=feature,
            id=id,
            input_tokens=input_tokens,
            member=member,
            model=model,
            on=on,
            outcome=outcome,
            output_tokens=output_tokens,
            provider=provider,
            tokens=tokens,
            action=action,
            detail=detail,
            member_color=member_color,
            target_id=target_id,
            target_kind=target_kind,
            user_id=user_id,
        )

        return ai_activity_row

