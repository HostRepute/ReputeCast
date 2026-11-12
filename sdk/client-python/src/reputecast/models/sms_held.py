from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_held_not_sent import SmsHeldNotSent





T = TypeVar("T", bound="SmsHeld")



@_attrs_define
class SmsHeld:
    """ 
        Attributes:
            lists (list[str]):
            no_consent (int):
            not_sent (SmsHeldNotSent):
            quiet_hours (int):
            skipped (int):
            skipped_countries (list[str]):
            suppressed (int):
            skip_reason (str | Unset):
            skip_sender (str | Unset):
     """

    lists: list[str]
    no_consent: int
    not_sent: SmsHeldNotSent
    quiet_hours: int
    skipped: int
    skipped_countries: list[str]
    suppressed: int
    skip_reason: str | Unset = UNSET
    skip_sender: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_held_not_sent import SmsHeldNotSent # noqa: PLC0415
        lists = self.lists



        no_consent = self.no_consent

        not_sent = self.not_sent.to_dict()

        quiet_hours = self.quiet_hours

        skipped = self.skipped

        skipped_countries = self.skipped_countries



        suppressed = self.suppressed

        skip_reason = self.skip_reason

        skip_sender = self.skip_sender


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "lists": lists,
            "noConsent": no_consent,
            "notSent": not_sent,
            "quietHours": quiet_hours,
            "skipped": skipped,
            "skippedCountries": skipped_countries,
            "suppressed": suppressed,
        })
        if skip_reason is not UNSET:
            field_dict["skipReason"] = skip_reason
        if skip_sender is not UNSET:
            field_dict["skipSender"] = skip_sender

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_held_not_sent import SmsHeldNotSent # noqa: PLC0415
        d = dict(src_dict)
        lists = cast(list[str], d.pop("lists"))


        no_consent = d.pop("noConsent")

        not_sent = SmsHeldNotSent.from_dict(d.pop("notSent"))




        quiet_hours = d.pop("quietHours")

        skipped = d.pop("skipped")

        skipped_countries = cast(list[str], d.pop("skippedCountries"))


        suppressed = d.pop("suppressed")

        skip_reason = d.pop("skipReason", UNSET)

        skip_sender = d.pop("skipSender", UNSET)

        sms_held = cls(
            lists=lists,
            no_consent=no_consent,
            not_sent=not_sent,
            quiet_hours=quiet_hours,
            skipped=skipped,
            skipped_countries=skipped_countries,
            suppressed=suppressed,
            skip_reason=skip_reason,
            skip_sender=skip_sender,
        )

        return sms_held

