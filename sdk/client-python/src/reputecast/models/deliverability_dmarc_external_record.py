from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.deliverability_dmarc_external_record_status import DeliverabilityDMARCExternalRecordStatus
from ..models.deliverability_dmarc_external_record_type import DeliverabilityDMARCExternalRecordType






T = TypeVar("T", bound="DeliverabilityDMARCExternalRecord")



@_attrs_define
class DeliverabilityDMARCExternalRecord:
    """ 
        Attributes:
            domain (str):
            name (str):
            status (DeliverabilityDMARCExternalRecordStatus):
            type_ (DeliverabilityDMARCExternalRecordType):
            value (str):
     """

    domain: str
    name: str
    status: DeliverabilityDMARCExternalRecordStatus
    type_: DeliverabilityDMARCExternalRecordType
    value: str





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        name = self.name

        status = self.status.value

        type_ = self.type_.value

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "name": name,
            "status": status,
            "type": type_,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        name = d.pop("name")

        status = DeliverabilityDMARCExternalRecordStatus(d.pop("status"))




        type_ = DeliverabilityDMARCExternalRecordType(d.pop("type"))




        value = d.pop("value")

        deliverability_dmarc_external_record = cls(
            domain=domain,
            name=name,
            status=status,
            type_=type_,
            value=value,
        )

        return deliverability_dmarc_external_record

