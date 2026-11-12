from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.report_language_row_reason import ReportLanguageRowReason
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="ReportLanguageRow")



@_attrs_define
class ReportLanguageRow:
    """ 
        Attributes:
            clicks (int):
            complaints (int):
            delivered (int):
            fallback (bool):
            human_opens (int):
            kinds (list[str]):
            language (str):
            name (str):
            readers (list[str]):
            sent (int):
            source (bool):
            unsubscribes (int):
            click_rate (float | Unset):
            complaint_rate (float | Unset):
            delivered_rate (float | Unset):
            human_open_rate (float | Unset):
            missing (str | Unset):
            reason (ReportLanguageRowReason | Unset):
            translated_from (int | Unset):
            unsubscribe_rate (float | Unset):
     """

    clicks: int
    complaints: int
    delivered: int
    fallback: bool
    human_opens: int
    kinds: list[str]
    language: str
    name: str
    readers: list[str]
    sent: int
    source: bool
    unsubscribes: int
    click_rate: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    delivered_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    missing: str | Unset = UNSET
    reason: ReportLanguageRowReason | Unset = UNSET
    translated_from: int | Unset = UNSET
    unsubscribe_rate: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        clicks = self.clicks

        complaints = self.complaints

        delivered = self.delivered

        fallback = self.fallback

        human_opens = self.human_opens

        kinds = self.kinds



        language = self.language

        name = self.name

        readers = self.readers



        sent = self.sent

        source = self.source

        unsubscribes = self.unsubscribes

        click_rate = self.click_rate

        complaint_rate = self.complaint_rate

        delivered_rate = self.delivered_rate

        human_open_rate = self.human_open_rate

        missing = self.missing

        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        translated_from = self.translated_from

        unsubscribe_rate = self.unsubscribe_rate


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "clicks": clicks,
            "complaints": complaints,
            "delivered": delivered,
            "fallback": fallback,
            "humanOpens": human_opens,
            "kinds": kinds,
            "language": language,
            "name": name,
            "readers": readers,
            "sent": sent,
            "source": source,
            "unsubscribes": unsubscribes,
        })
        if click_rate is not UNSET:
            field_dict["clickRate"] = click_rate
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if delivered_rate is not UNSET:
            field_dict["deliveredRate"] = delivered_rate
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if missing is not UNSET:
            field_dict["missing"] = missing
        if reason is not UNSET:
            field_dict["reason"] = reason
        if translated_from is not UNSET:
            field_dict["translatedFrom"] = translated_from
        if unsubscribe_rate is not UNSET:
            field_dict["unsubscribeRate"] = unsubscribe_rate

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        clicks = d.pop("clicks")

        complaints = d.pop("complaints")

        delivered = d.pop("delivered")

        fallback = d.pop("fallback")

        human_opens = d.pop("humanOpens")

        kinds = cast(list[str], d.pop("kinds"))


        language = d.pop("language")

        name = d.pop("name")

        readers = cast(list[str], d.pop("readers"))


        sent = d.pop("sent")

        source = d.pop("source")

        unsubscribes = d.pop("unsubscribes")

        click_rate = d.pop("clickRate", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        delivered_rate = d.pop("deliveredRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        missing = d.pop("missing", UNSET)

        _reason = d.pop("reason", UNSET)
        reason: ReportLanguageRowReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = ReportLanguageRowReason(_reason)




        translated_from = d.pop("translatedFrom", UNSET)

        unsubscribe_rate = d.pop("unsubscribeRate", UNSET)

        report_language_row = cls(
            clicks=clicks,
            complaints=complaints,
            delivered=delivered,
            fallback=fallback,
            human_opens=human_opens,
            kinds=kinds,
            language=language,
            name=name,
            readers=readers,
            sent=sent,
            source=source,
            unsubscribes=unsubscribes,
            click_rate=click_rate,
            complaint_rate=complaint_rate,
            delivered_rate=delivered_rate,
            human_open_rate=human_open_rate,
            missing=missing,
            reason=reason,
            translated_from=translated_from,
            unsubscribe_rate=unsubscribe_rate,
        )

        return report_language_row

