from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_postmaster_via import DeliverabilityPostmasterVia
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_postmaster_peak import DeliverabilityPostmasterPeak
  from ..models.deliverability_postmaster_point import DeliverabilityPostmasterPoint





T = TypeVar("T", bound="DeliverabilityPostmaster")



@_attrs_define
class DeliverabilityPostmaster:
    """ 
        Attributes:
            domain (str):
            from_ (datetime.date):
            series (list[DeliverabilityPostmasterPoint]):
            stale (bool):
            to (datetime.date):
            via (DeliverabilityPostmasterVia):
            dkim (float | Unset):
            dmarc (float | Unset):
            feedback_loop_flagged (int | Unset):
            peak (DeliverabilityPostmasterPeak | Unset):
            spam_rate (float | Unset):
            spf (float | Unset):
            tls (float | Unset):
     """

    domain: str
    from_: datetime.date
    series: list[DeliverabilityPostmasterPoint]
    stale: bool
    to: datetime.date
    via: DeliverabilityPostmasterVia
    dkim: float | Unset = UNSET
    dmarc: float | Unset = UNSET
    feedback_loop_flagged: int | Unset = UNSET
    peak: DeliverabilityPostmasterPeak | Unset = UNSET
    spam_rate: float | Unset = UNSET
    spf: float | Unset = UNSET
    tls: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_postmaster_peak import DeliverabilityPostmasterPeak # noqa: PLC0415
        from ..models.deliverability_postmaster_point import DeliverabilityPostmasterPoint # noqa: PLC0415
        domain = self.domain

        from_ = self.from_.isoformat()

        series = []
        for series_item_data in self.series:
            series_item = series_item_data.to_dict()
            series.append(series_item)



        stale = self.stale

        to = self.to.isoformat()

        via = self.via.value

        dkim = self.dkim

        dmarc = self.dmarc

        feedback_loop_flagged = self.feedback_loop_flagged

        peak: dict[str, Any] | Unset = UNSET
        if not isinstance(self.peak, Unset):
            peak = self.peak.to_dict()

        spam_rate = self.spam_rate

        spf = self.spf

        tls = self.tls


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "from": from_,
            "series": series,
            "stale": stale,
            "to": to,
            "via": via,
        })
        if dkim is not UNSET:
            field_dict["dkim"] = dkim
        if dmarc is not UNSET:
            field_dict["dmarc"] = dmarc
        if feedback_loop_flagged is not UNSET:
            field_dict["feedbackLoopFlagged"] = feedback_loop_flagged
        if peak is not UNSET:
            field_dict["peak"] = peak
        if spam_rate is not UNSET:
            field_dict["spamRate"] = spam_rate
        if spf is not UNSET:
            field_dict["spf"] = spf
        if tls is not UNSET:
            field_dict["tls"] = tls

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_postmaster_peak import DeliverabilityPostmasterPeak # noqa: PLC0415
        from ..models.deliverability_postmaster_point import DeliverabilityPostmasterPoint # noqa: PLC0415
        d = dict(src_dict)
        domain = d.pop("domain")

        from_ = datetime.date.fromisoformat(d.pop("from"))




        series = []
        _series = d.pop("series")
        for series_item_data in (_series):
            series_item = DeliverabilityPostmasterPoint.from_dict(series_item_data)



            series.append(series_item)


        stale = d.pop("stale")

        to = datetime.date.fromisoformat(d.pop("to"))




        via = DeliverabilityPostmasterVia(d.pop("via"))




        dkim = d.pop("dkim", UNSET)

        dmarc = d.pop("dmarc", UNSET)

        feedback_loop_flagged = d.pop("feedbackLoopFlagged", UNSET)

        _peak = d.pop("peak", UNSET)
        peak: DeliverabilityPostmasterPeak | Unset
        if isinstance(_peak,  Unset):
            peak = UNSET
        else:
            peak = DeliverabilityPostmasterPeak.from_dict(_peak)




        spam_rate = d.pop("spamRate", UNSET)

        spf = d.pop("spf", UNSET)

        tls = d.pop("tls", UNSET)

        deliverability_postmaster = cls(
            domain=domain,
            from_=from_,
            series=series,
            stale=stale,
            to=to,
            via=via,
            dkim=dkim,
            dmarc=dmarc,
            feedback_loop_flagged=feedback_loop_flagged,
            peak=peak,
            spam_rate=spam_rate,
            spf=spf,
            tls=tls,
        )

        return deliverability_postmaster

