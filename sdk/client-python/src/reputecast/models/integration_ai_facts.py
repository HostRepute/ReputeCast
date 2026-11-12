from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.integration_ai_facts_cap_kind import IntegrationAIFactsCapKind
from ..types import UNSET, Unset






T = TypeVar("T", bound="IntegrationAIFacts")



@_attrs_define
class IntegrationAIFacts:
    """ 
        Attributes:
            cap_tokens (int):
            cap_usd (float):
            cost_micros (int):
            key_rejected (bool):
            share (float): Share of the cap used, 0 to 1
            tokens (int):
            cap_kind (IntegrationAIFactsCapKind | Unset):
            connected (str | Unset):
            model_name (str | Unset):
     """

    cap_tokens: int
    cap_usd: float
    cost_micros: int
    key_rejected: bool
    share: float
    tokens: int
    cap_kind: IntegrationAIFactsCapKind | Unset = UNSET
    connected: str | Unset = UNSET
    model_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        cap_tokens = self.cap_tokens

        cap_usd = self.cap_usd

        cost_micros = self.cost_micros

        key_rejected = self.key_rejected

        share = self.share

        tokens = self.tokens

        cap_kind: str | Unset = UNSET
        if not isinstance(self.cap_kind, Unset):
            cap_kind = self.cap_kind.value


        connected = self.connected

        model_name = self.model_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capTokens": cap_tokens,
            "capUsd": cap_usd,
            "costMicros": cost_micros,
            "keyRejected": key_rejected,
            "share": share,
            "tokens": tokens,
        })
        if cap_kind is not UNSET:
            field_dict["capKind"] = cap_kind
        if connected is not UNSET:
            field_dict["connected"] = connected
        if model_name is not UNSET:
            field_dict["modelName"] = model_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        cap_tokens = d.pop("capTokens")

        cap_usd = d.pop("capUsd")

        cost_micros = d.pop("costMicros")

        key_rejected = d.pop("keyRejected")

        share = d.pop("share")

        tokens = d.pop("tokens")

        _cap_kind = d.pop("capKind", UNSET)
        cap_kind: IntegrationAIFactsCapKind | Unset
        if isinstance(_cap_kind,  Unset):
            cap_kind = UNSET
        else:
            cap_kind = IntegrationAIFactsCapKind(_cap_kind)




        connected = d.pop("connected", UNSET)

        model_name = d.pop("modelName", UNSET)

        integration_ai_facts = cls(
            cap_tokens=cap_tokens,
            cap_usd=cap_usd,
            cost_micros=cost_micros,
            key_rejected=key_rejected,
            share=share,
            tokens=tokens,
            cap_kind=cap_kind,
            connected=connected,
            model_name=model_name,
        )

        return integration_ai_facts

