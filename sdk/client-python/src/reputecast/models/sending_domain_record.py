from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sending_domain_record_purpose import SendingDomainRecordPurpose
from ..models.sending_domain_record_status import SendingDomainRecordStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SendingDomainRecord")



@_attrs_define
class SendingDomainRecord:
    """ 
        Attributes:
            key (str):
            label (str):
            name (str):
            note (str):
            provider (str):
            provider_id (str):
            purpose (SendingDomainRecordPurpose):
            status (SendingDomainRecordStatus):
            status_text (str):
            type_ (str):
            value (str):
            zone_line (str):
            checked_at (datetime.datetime | Unset):
            found (str | Unset):
     """

    key: str
    label: str
    name: str
    note: str
    provider: str
    provider_id: str
    purpose: SendingDomainRecordPurpose
    status: SendingDomainRecordStatus
    status_text: str
    type_: str
    value: str
    zone_line: str
    checked_at: datetime.datetime | Unset = UNSET
    found: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        key = self.key

        label = self.label

        name = self.name

        note = self.note

        provider = self.provider

        provider_id = self.provider_id

        purpose = self.purpose.value

        status = self.status.value

        status_text = self.status_text

        type_ = self.type_

        value = self.value

        zone_line = self.zone_line

        checked_at: str | Unset = UNSET
        if not isinstance(self.checked_at, Unset):
            checked_at = self.checked_at.isoformat()

        found = self.found


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "key": key,
            "label": label,
            "name": name,
            "note": note,
            "provider": provider,
            "providerId": provider_id,
            "purpose": purpose,
            "status": status,
            "statusText": status_text,
            "type": type_,
            "value": value,
            "zoneLine": zone_line,
        })
        if checked_at is not UNSET:
            field_dict["checkedAt"] = checked_at
        if found is not UNSET:
            field_dict["found"] = found

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        key = d.pop("key")

        label = d.pop("label")

        name = d.pop("name")

        note = d.pop("note")

        provider = d.pop("provider")

        provider_id = d.pop("providerId")

        purpose = SendingDomainRecordPurpose(d.pop("purpose"))




        status = SendingDomainRecordStatus(d.pop("status"))




        status_text = d.pop("statusText")

        type_ = d.pop("type")

        value = d.pop("value")

        zone_line = d.pop("zoneLine")

        _checked_at = d.pop("checkedAt", UNSET)
        checked_at: datetime.datetime | Unset
        if isinstance(_checked_at,  Unset):
            checked_at = UNSET
        else:
            checked_at = datetime.datetime.fromisoformat(_checked_at)




        found = d.pop("found", UNSET)

        sending_domain_record = cls(
            key=key,
            label=label,
            name=name,
            note=note,
            provider=provider,
            provider_id=provider_id,
            purpose=purpose,
            status=status,
            status_text=status_text,
            type_=type_,
            value=value,
            zone_line=zone_line,
            checked_at=checked_at,
            found=found,
        )

        return sending_domain_record

