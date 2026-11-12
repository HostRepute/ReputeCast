from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_snds_filter_result import DeliverabilitySNDSFilterResult
from ..models.deliverability_snds_via import DeliverabilitySNDSVia
from ..types import UNSET, Unset






T = TypeVar("T", bound="DeliverabilitySNDS")



@_attrs_define
class DeliverabilitySNDS:
    """ 
        Attributes:
            filter_result (DeliverabilitySNDSFilterResult):
            ip (str):
            jmrp_reports (int):
            messages7d (int):
            stale (bool):
            trap_hits (int):
            via (DeliverabilitySNDSVia):
            complaint_rate (float | Unset):
     """

    filter_result: DeliverabilitySNDSFilterResult
    ip: str
    jmrp_reports: int
    messages7d: int
    stale: bool
    trap_hits: int
    via: DeliverabilitySNDSVia
    complaint_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        filter_result = self.filter_result.value

        ip = self.ip

        jmrp_reports = self.jmrp_reports

        messages7d = self.messages7d

        stale = self.stale

        trap_hits = self.trap_hits

        via = self.via.value

        complaint_rate = self.complaint_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "filterResult": filter_result,
            "ip": ip,
            "jmrpReports": jmrp_reports,
            "messages7d": messages7d,
            "stale": stale,
            "trapHits": trap_hits,
            "via": via,
        })
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        filter_result = DeliverabilitySNDSFilterResult(d.pop("filterResult"))




        ip = d.pop("ip")

        jmrp_reports = d.pop("jmrpReports")

        messages7d = d.pop("messages7d")

        stale = d.pop("stale")

        trap_hits = d.pop("trapHits")

        via = DeliverabilitySNDSVia(d.pop("via"))




        complaint_rate = d.pop("complaintRate", UNSET)

        deliverability_snds = cls(
            filter_result=filter_result,
            ip=ip,
            jmrp_reports=jmrp_reports,
            messages7d=messages7d,
            stale=stale,
            trap_hits=trap_hits,
            via=via,
            complaint_rate=complaint_rate,
        )

        return deliverability_snds

