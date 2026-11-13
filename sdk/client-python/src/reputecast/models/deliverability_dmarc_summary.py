from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_dmarc_summary_via import DeliverabilityDMARCSummaryVia
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.deliverability_dmarc_collect import DeliverabilityDMARCCollect
  from ..models.deliverability_dmarc_source import DeliverabilityDMARCSource





T = TypeVar("T", bound="DeliverabilityDMARCSummary")



@_attrs_define
class DeliverabilityDMARCSummary:
    """ 
        Attributes:
            days (int):
            messages (int):
            reports (int):
            sources (list[DeliverabilityDMARCSource]):
            via (DeliverabilityDMARCSummaryVia):
            collect (DeliverabilityDMARCCollect | Unset):
            domain (str | Unset):
            pass_rate (float | Unset):
            policy (str | Unset):
            rua (str | Unset):
            unknown (DeliverabilityDMARCSource | Unset):
     """

    days: int
    messages: int
    reports: int
    sources: list[DeliverabilityDMARCSource]
    via: DeliverabilityDMARCSummaryVia
    collect: DeliverabilityDMARCCollect | Unset = UNSET
    domain: str | Unset = UNSET
    pass_rate: float | Unset = UNSET
    policy: str | Unset = UNSET
    rua: str | Unset = UNSET
    unknown: DeliverabilityDMARCSource | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_dmarc_collect import DeliverabilityDMARCCollect # noqa: PLC0415
        from ..models.deliverability_dmarc_source import DeliverabilityDMARCSource # noqa: PLC0415
        days = self.days

        messages = self.messages

        reports = self.reports

        sources = []
        for sources_item_data in self.sources:
            sources_item = sources_item_data.to_dict()
            sources.append(sources_item)



        via = self.via.value

        collect: dict[str, Any] | Unset = UNSET
        if not isinstance(self.collect, Unset):
            collect = self.collect.to_dict()

        domain = self.domain

        pass_rate = self.pass_rate

        policy = self.policy

        rua = self.rua

        unknown: dict[str, Any] | Unset = UNSET
        if not isinstance(self.unknown, Unset):
            unknown = self.unknown.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "messages": messages,
            "reports": reports,
            "sources": sources,
            "via": via,
        })
        if collect is not UNSET:
            field_dict["collect"] = collect
        if domain is not UNSET:
            field_dict["domain"] = domain
        if pass_rate is not UNSET:
            field_dict["passRate"] = pass_rate
        if policy is not UNSET:
            field_dict["policy"] = policy
        if rua is not UNSET:
            field_dict["rua"] = rua
        if unknown is not UNSET:
            field_dict["unknown"] = unknown

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_dmarc_collect import DeliverabilityDMARCCollect # noqa: PLC0415
        from ..models.deliverability_dmarc_source import DeliverabilityDMARCSource # noqa: PLC0415
        d = dict(src_dict)
        days = d.pop("days")

        messages = d.pop("messages")

        reports = d.pop("reports")

        sources = []
        _sources = d.pop("sources")
        for sources_item_data in (_sources):
            sources_item = DeliverabilityDMARCSource.from_dict(sources_item_data)



            sources.append(sources_item)


        via = DeliverabilityDMARCSummaryVia(d.pop("via"))




        _collect = d.pop("collect", UNSET)
        collect: DeliverabilityDMARCCollect | Unset
        if isinstance(_collect,  Unset):
            collect = UNSET
        else:
            collect = DeliverabilityDMARCCollect.from_dict(_collect)




        domain = d.pop("domain", UNSET)

        pass_rate = d.pop("passRate", UNSET)

        policy = d.pop("policy", UNSET)

        rua = d.pop("rua", UNSET)

        _unknown = d.pop("unknown", UNSET)
        unknown: DeliverabilityDMARCSource | Unset
        if isinstance(_unknown,  Unset):
            unknown = UNSET
        else:
            unknown = DeliverabilityDMARCSource.from_dict(_unknown)




        deliverability_dmarc_summary = cls(
            days=days,
            messages=messages,
            reports=reports,
            sources=sources,
            via=via,
            collect=collect,
            domain=domain,
            pass_rate=pass_rate,
            policy=policy,
            rua=rua,
            unknown=unknown,
        )

        return deliverability_dmarc_summary

