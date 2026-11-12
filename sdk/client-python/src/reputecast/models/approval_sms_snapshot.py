from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.approval_sms_country import ApprovalSMSCountry





T = TypeVar("T", bound="ApprovalSMSSnapshot")



@_attrs_define
class ApprovalSMSSnapshot:
    """ 
        Attributes:
            body (str):
            chars (int):
            cost (float):
            countries (list[ApprovalSMSCountry]):
            encoding (str):
            opt_out (str):
            prefix (str):
            recipients (int):
            schedule (str):
            segments (int):
            text (str):
     """

    body: str
    chars: int
    cost: float
    countries: list[ApprovalSMSCountry]
    encoding: str
    opt_out: str
    prefix: str
    recipients: int
    schedule: str
    segments: int
    text: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.approval_sms_country import ApprovalSMSCountry # noqa: PLC0415
        body = self.body

        chars = self.chars

        cost = self.cost

        countries = []
        for countries_item_data in self.countries:
            countries_item = countries_item_data.to_dict()
            countries.append(countries_item)



        encoding = self.encoding

        opt_out = self.opt_out

        prefix = self.prefix

        recipients = self.recipients

        schedule = self.schedule

        segments = self.segments

        text = self.text


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "body": body,
            "chars": chars,
            "cost": cost,
            "countries": countries,
            "encoding": encoding,
            "optOut": opt_out,
            "prefix": prefix,
            "recipients": recipients,
            "schedule": schedule,
            "segments": segments,
            "text": text,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.approval_sms_country import ApprovalSMSCountry # noqa: PLC0415
        d = dict(src_dict)
        body = d.pop("body")

        chars = d.pop("chars")

        cost = d.pop("cost")

        countries = []
        _countries = d.pop("countries")
        for countries_item_data in (_countries):
            countries_item = ApprovalSMSCountry.from_dict(countries_item_data)



            countries.append(countries_item)


        encoding = d.pop("encoding")

        opt_out = d.pop("optOut")

        prefix = d.pop("prefix")

        recipients = d.pop("recipients")

        schedule = d.pop("schedule")

        segments = d.pop("segments")

        text = d.pop("text")

        approval_sms_snapshot = cls(
            body=body,
            chars=chars,
            cost=cost,
            countries=countries,
            encoding=encoding,
            opt_out=opt_out,
            prefix=prefix,
            recipients=recipients,
            schedule=schedule,
            segments=segments,
            text=text,
        )

        return approval_sms_snapshot

