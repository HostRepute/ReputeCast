from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_dmarc_source_result import DeliverabilityDMARCSourceResult
from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilityDMARCSource")



@_attrs_define
class DeliverabilityDMARCSource:
    """ 
        Attributes:
            dkim (str):
            known (bool):
            label (str): Amazon SES, Postal · 203.0.113.25, Unknown · 198.51.100.77
            messages (int):
            result (DeliverabilityDMARCSourceResult):
            spf (str): aligned, or the raw result: fail, softfail, none…
            adapter (str | Unset):
            ip (str | Unset):
            provider_id (str | Unset):
     """

    dkim: str
    known: bool
    label: str
    messages: int
    result: DeliverabilityDMARCSourceResult
    spf: str
    adapter: str | Unset = UNSET
    ip: str | Unset = UNSET
    provider_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        dkim = self.dkim

        known = self.known

        label = self.label

        messages = self.messages

        result = self.result.value

        spf = self.spf

        adapter = self.adapter

        ip = self.ip

        provider_id = self.provider_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "dkim": dkim,
            "known": known,
            "label": label,
            "messages": messages,
            "result": result,
            "spf": spf,
        })
        if adapter is not UNSET:
            field_dict["adapter"] = adapter
        if ip is not UNSET:
            field_dict["ip"] = ip
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        dkim = d.pop("dkim")

        known = d.pop("known")

        label = d.pop("label")

        messages = d.pop("messages")

        result = DeliverabilityDMARCSourceResult(d.pop("result"))




        spf = d.pop("spf")

        adapter = d.pop("adapter", UNSET)

        ip = d.pop("ip", UNSET)

        provider_id = d.pop("providerId", UNSET)

        deliverability_dmarc_source = cls(
            dkim=dkim,
            known=known,
            label=label,
            messages=messages,
            result=result,
            spf=spf,
            adapter=adapter,
            ip=ip,
            provider_id=provider_id,
        )

        return deliverability_dmarc_source

