from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.growth_source_row_too_few import GrowthSourceRowTooFew
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.growth_source import GrowthSource





T = TypeVar("T", bound="GrowthSourceRow")



@_attrs_define
class GrowthSourceRow:
    """ 
        Attributes:
            contacts (int):
            delivered (int):
            left (int):
            sent (int):
            source (GrowthSource):
            bounce_flag (float | Unset):
            bounce_rate (float | Unset):
            complaint_flag (float | Unset):
            complaint_rate (float | Unset):
            human_open_rate (float | Unset):
            joined (int | Unset):
            net (int | Unset):
            too_few (GrowthSourceRowTooFew | Unset):
     """

    contacts: int
    delivered: int
    left: int
    sent: int
    source: GrowthSource
    bounce_flag: float | Unset = UNSET
    bounce_rate: float | Unset = UNSET
    complaint_flag: float | Unset = UNSET
    complaint_rate: float | Unset = UNSET
    human_open_rate: float | Unset = UNSET
    joined: int | Unset = UNSET
    net: int | Unset = UNSET
    too_few: GrowthSourceRowTooFew | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        contacts = self.contacts

        delivered = self.delivered

        left = self.left

        sent = self.sent

        source = self.source.to_dict()

        bounce_flag = self.bounce_flag

        bounce_rate = self.bounce_rate

        complaint_flag = self.complaint_flag

        complaint_rate = self.complaint_rate

        human_open_rate = self.human_open_rate

        joined = self.joined

        net = self.net

        too_few: str | Unset = UNSET
        if not isinstance(self.too_few, Unset):
            too_few = self.too_few.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "delivered": delivered,
            "left": left,
            "sent": sent,
            "source": source,
        })
        if bounce_flag is not UNSET:
            field_dict["bounceFlag"] = bounce_flag
        if bounce_rate is not UNSET:
            field_dict["bounceRate"] = bounce_rate
        if complaint_flag is not UNSET:
            field_dict["complaintFlag"] = complaint_flag
        if complaint_rate is not UNSET:
            field_dict["complaintRate"] = complaint_rate
        if human_open_rate is not UNSET:
            field_dict["humanOpenRate"] = human_open_rate
        if joined is not UNSET:
            field_dict["joined"] = joined
        if net is not UNSET:
            field_dict["net"] = net
        if too_few is not UNSET:
            field_dict["tooFew"] = too_few

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.growth_source import GrowthSource # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        delivered = d.pop("delivered")

        left = d.pop("left")

        sent = d.pop("sent")

        source = GrowthSource.from_dict(d.pop("source"))




        bounce_flag = d.pop("bounceFlag", UNSET)

        bounce_rate = d.pop("bounceRate", UNSET)

        complaint_flag = d.pop("complaintFlag", UNSET)

        complaint_rate = d.pop("complaintRate", UNSET)

        human_open_rate = d.pop("humanOpenRate", UNSET)

        joined = d.pop("joined", UNSET)

        net = d.pop("net", UNSET)

        _too_few = d.pop("tooFew", UNSET)
        too_few: GrowthSourceRowTooFew | Unset
        if isinstance(_too_few,  Unset):
            too_few = UNSET
        else:
            too_few = GrowthSourceRowTooFew(_too_few)




        growth_source_row = cls(
            contacts=contacts,
            delivered=delivered,
            left=left,
            sent=sent,
            source=source,
            bounce_flag=bounce_flag,
            bounce_rate=bounce_rate,
            complaint_flag=complaint_flag,
            complaint_rate=complaint_rate,
            human_open_rate=human_open_rate,
            joined=joined,
            net=net,
            too_few=too_few,
        )

        return growth_source_row

