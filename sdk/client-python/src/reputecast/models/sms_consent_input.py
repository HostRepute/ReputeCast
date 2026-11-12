from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_consent_input_status import SmsConsentInputStatus
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID

if TYPE_CHECKING:
  from ..models.sms_consent_evidence import SmsConsentEvidence





T = TypeVar("T", bound="SmsConsentInput")



@_attrs_define
class SmsConsentInput:
    """ 
        Attributes:
            list_id (UUID):
            status (SmsConsentInputStatus):
            evidence (SmsConsentEvidence | Unset):
            note (str | Unset):
     """

    list_id: UUID
    status: SmsConsentInputStatus
    evidence: SmsConsentEvidence | Unset = UNSET
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_consent_evidence import SmsConsentEvidence # noqa: PLC0415
        list_id = str(self.list_id)

        status = self.status.value

        evidence: dict[str, Any] | Unset = UNSET
        if not isinstance(self.evidence, Unset):
            evidence = self.evidence.to_dict()

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listId": list_id,
            "status": status,
        })
        if evidence is not UNSET:
            field_dict["evidence"] = evidence
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_consent_evidence import SmsConsentEvidence # noqa: PLC0415
        d = dict(src_dict)
        list_id = UUID(d.pop("listId"))




        status = SmsConsentInputStatus(d.pop("status"))




        _evidence = d.pop("evidence", UNSET)
        evidence: SmsConsentEvidence | Unset
        if isinstance(_evidence,  Unset):
            evidence = UNSET
        else:
            evidence = SmsConsentEvidence.from_dict(_evidence)




        note = d.pop("note", UNSET)

        sms_consent_input = cls(
            list_id=list_id,
            status=status,
            evidence=evidence,
            note=note,
        )

        return sms_consent_input

