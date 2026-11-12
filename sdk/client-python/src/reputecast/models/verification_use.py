from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_use_kind import VerificationUseKind
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.verification_form_count import VerificationFormCount





T = TypeVar("T", bound="VerificationUse")



@_attrs_define
class VerificationUse:
    """ 
        Attributes:
            addresses (int):
            at (datetime.datetime):
            by (str):
            cost (float):
            forms (list[VerificationFormCount]):
            id (str):
            invalid (int):
            kind (VerificationUseKind):
            label (str):
            no_answer (int):
            risky (int):
            service (str):
            service_name (str):
            status (str):
            suppressed (int):
            unknown (int):
            valid (int):
            finished_at (datetime.datetime | Unset):
     """

    addresses: int
    at: datetime.datetime
    by: str
    cost: float
    forms: list[VerificationFormCount]
    id: str
    invalid: int
    kind: VerificationUseKind
    label: str
    no_answer: int
    risky: int
    service: str
    service_name: str
    status: str
    suppressed: int
    unknown: int
    valid: int
    finished_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_form_count import VerificationFormCount # noqa: PLC0415
        addresses = self.addresses

        at = self.at.isoformat()

        by = self.by

        cost = self.cost

        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        id = self.id

        invalid = self.invalid

        kind = self.kind.value

        label = self.label

        no_answer = self.no_answer

        risky = self.risky

        service = self.service

        service_name = self.service_name

        status = self.status

        suppressed = self.suppressed

        unknown = self.unknown

        valid = self.valid

        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addresses": addresses,
            "at": at,
            "by": by,
            "cost": cost,
            "forms": forms,
            "id": id,
            "invalid": invalid,
            "kind": kind,
            "label": label,
            "noAnswer": no_answer,
            "risky": risky,
            "service": service,
            "serviceName": service_name,
            "status": status,
            "suppressed": suppressed,
            "unknown": unknown,
            "valid": valid,
        })
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_form_count import VerificationFormCount # noqa: PLC0415
        d = dict(src_dict)
        addresses = d.pop("addresses")

        at = datetime.datetime.fromisoformat(d.pop("at"))




        by = d.pop("by")

        cost = d.pop("cost")

        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = VerificationFormCount.from_dict(forms_item_data)



            forms.append(forms_item)


        id = d.pop("id")

        invalid = d.pop("invalid")

        kind = VerificationUseKind(d.pop("kind"))




        label = d.pop("label")

        no_answer = d.pop("noAnswer")

        risky = d.pop("risky")

        service = d.pop("service")

        service_name = d.pop("serviceName")

        status = d.pop("status")

        suppressed = d.pop("suppressed")

        unknown = d.pop("unknown")

        valid = d.pop("valid")

        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        verification_use = cls(
            addresses=addresses,
            at=at,
            by=by,
            cost=cost,
            forms=forms,
            id=id,
            invalid=invalid,
            kind=kind,
            label=label,
            no_answer=no_answer,
            risky=risky,
            service=service,
            service_name=service_name,
            status=status,
            suppressed=suppressed,
            unknown=unknown,
            valid=valid,
            finished_at=finished_at,
        )

        return verification_use

