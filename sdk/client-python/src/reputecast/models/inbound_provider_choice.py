from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.inbound_provider_choice_inbound import InboundProviderChoiceInbound
from ..types import UNSET, Unset






T = TypeVar("T", bound="InboundProviderChoice")



@_attrs_define
class InboundProviderChoice:
    """ 
        Attributes:
            adapter (str):
            id (str):
            inbound (InboundProviderChoiceInbound):
            name (str):
            reason (str | Unset):
     """

    adapter: str
    id: str
    inbound: InboundProviderChoiceInbound
    name: str
    reason: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        adapter = self.adapter

        id = self.id

        inbound = self.inbound.value

        name = self.name

        reason = self.reason


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "adapter": adapter,
            "id": id,
            "inbound": inbound,
            "name": name,
        })
        if reason is not UNSET:
            field_dict["reason"] = reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        adapter = d.pop("adapter")

        id = d.pop("id")

        inbound = InboundProviderChoiceInbound(d.pop("inbound"))




        name = d.pop("name")

        reason = d.pop("reason", UNSET)

        inbound_provider_choice = cls(
            adapter=adapter,
            id=id,
            inbound=inbound,
            name=name,
            reason=reason,
        )

        return inbound_provider_choice

