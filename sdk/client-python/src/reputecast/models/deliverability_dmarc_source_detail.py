from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityDMARCSourceDetail")



@_attrs_define
class DeliverabilityDMARCSourceDetail:
    """ 
        Attributes:
            disposition (str):
            dkim (str):
            dkim_domains (list[str]):
            first_seen (datetime.datetime):
            header_from (list[str]):
            ip (str):
            known (bool):
            last_seen (datetime.datetime):
            messages (int):
            reporters (list[str]):
            reports (int):
            spf (str):
            spf_domains (list[str]):
            policy (str | Unset):
            provider (str | Unset):
            ptr (str | Unset):
     """

    disposition: str
    dkim: str
    dkim_domains: list[str]
    first_seen: datetime.datetime
    header_from: list[str]
    ip: str
    known: bool
    last_seen: datetime.datetime
    messages: int
    reporters: list[str]
    reports: int
    spf: str
    spf_domains: list[str]
    policy: str | Unset = UNSET
    provider: str | Unset = UNSET
    ptr: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        disposition = self.disposition

        dkim = self.dkim

        dkim_domains = self.dkim_domains



        first_seen = self.first_seen.isoformat()

        header_from = self.header_from



        ip = self.ip

        known = self.known

        last_seen = self.last_seen.isoformat()

        messages = self.messages

        reporters = self.reporters



        reports = self.reports

        spf = self.spf

        spf_domains = self.spf_domains



        policy = self.policy

        provider = self.provider

        ptr = self.ptr


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "disposition": disposition,
            "dkim": dkim,
            "dkimDomains": dkim_domains,
            "firstSeen": first_seen,
            "headerFrom": header_from,
            "ip": ip,
            "known": known,
            "lastSeen": last_seen,
            "messages": messages,
            "reporters": reporters,
            "reports": reports,
            "spf": spf,
            "spfDomains": spf_domains,
        })
        if policy is not UNSET:
            field_dict["policy"] = policy
        if provider is not UNSET:
            field_dict["provider"] = provider
        if ptr is not UNSET:
            field_dict["ptr"] = ptr

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        disposition = d.pop("disposition")

        dkim = d.pop("dkim")

        dkim_domains = cast(list[str], d.pop("dkimDomains"))


        first_seen = datetime.datetime.fromisoformat(d.pop("firstSeen"))




        header_from = cast(list[str], d.pop("headerFrom"))


        ip = d.pop("ip")

        known = d.pop("known")

        last_seen = datetime.datetime.fromisoformat(d.pop("lastSeen"))




        messages = d.pop("messages")

        reporters = cast(list[str], d.pop("reporters"))


        reports = d.pop("reports")

        spf = d.pop("spf")

        spf_domains = cast(list[str], d.pop("spfDomains"))


        policy = d.pop("policy", UNSET)

        provider = d.pop("provider", UNSET)

        ptr = d.pop("ptr", UNSET)

        deliverability_dmarc_source_detail = cls(
            disposition=disposition,
            dkim=dkim,
            dkim_domains=dkim_domains,
            first_seen=first_seen,
            header_from=header_from,
            ip=ip,
            known=known,
            last_seen=last_seen,
            messages=messages,
            reporters=reporters,
            reports=reports,
            spf=spf,
            spf_domains=spf_domains,
            policy=policy,
            provider=provider,
            ptr=ptr,
        )

        return deliverability_dmarc_source_detail

