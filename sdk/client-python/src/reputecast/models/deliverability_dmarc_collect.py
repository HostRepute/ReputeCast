from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.deliverability_dmarc_external_record import DeliverabilityDMARCExternalRecord





T = TypeVar("T", bound="DeliverabilityDMARCCollect")



@_attrs_define
class DeliverabilityDMARCCollect:
    """ 
        Attributes:
            address (str): dmarc@reply.news.acme.io, for the rua of DMARC records
            external_records (list[DeliverabilityDMARCExternalRecord]):
            last_received_at (datetime.datetime | Unset):
     """

    address: str
    external_records: list[DeliverabilityDMARCExternalRecord]
    last_received_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.deliverability_dmarc_external_record import DeliverabilityDMARCExternalRecord # noqa: PLC0415
        address = self.address

        external_records = []
        for external_records_item_data in self.external_records:
            external_records_item = external_records_item_data.to_dict()
            external_records.append(external_records_item)



        last_received_at: str | Unset = UNSET
        if not isinstance(self.last_received_at, Unset):
            last_received_at = self.last_received_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "address": address,
            "externalRecords": external_records,
        })
        if last_received_at is not UNSET:
            field_dict["lastReceivedAt"] = last_received_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.deliverability_dmarc_external_record import DeliverabilityDMARCExternalRecord # noqa: PLC0415
        d = dict(src_dict)
        address = d.pop("address")

        external_records = []
        _external_records = d.pop("externalRecords")
        for external_records_item_data in (_external_records):
            external_records_item = DeliverabilityDMARCExternalRecord.from_dict(external_records_item_data)



            external_records.append(external_records_item)


        _last_received_at = d.pop("lastReceivedAt", UNSET)
        last_received_at: datetime.datetime | Unset
        if isinstance(_last_received_at,  Unset):
            last_received_at = UNSET
        else:
            last_received_at = datetime.datetime.fromisoformat(_last_received_at)




        deliverability_dmarc_collect = cls(
            address=address,
            external_records=external_records,
            last_received_at=last_received_at,
        )

        return deliverability_dmarc_collect

