from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="DeliverabilityDMARCUpload")



@_attrs_define
class DeliverabilityDMARCUpload:
    """ 
        Attributes:
            domain (str):
            from_ (datetime.datetime):
            messages (int):
            org_name (str):
            records (int):
            report_id (str):
            to (datetime.datetime):
     """

    domain: str
    from_: datetime.datetime
    messages: int
    org_name: str
    records: int
    report_id: str
    to: datetime.datetime





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        from_ = self.from_.isoformat()

        messages = self.messages

        org_name = self.org_name

        records = self.records

        report_id = self.report_id

        to = self.to.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "from": from_,
            "messages": messages,
            "orgName": org_name,
            "records": records,
            "reportId": report_id,
            "to": to,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        from_ = datetime.datetime.fromisoformat(d.pop("from"))




        messages = d.pop("messages")

        org_name = d.pop("orgName")

        records = d.pop("records")

        report_id = d.pop("reportId")

        to = datetime.datetime.fromisoformat(d.pop("to"))




        deliverability_dmarc_upload = cls(
            domain=domain,
            from_=from_,
            messages=messages,
            org_name=org_name,
            records=records,
            report_id=report_id,
            to=to,
        )

        return deliverability_dmarc_upload

