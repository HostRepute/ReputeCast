from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_found_sender_kind import SmsFoundSenderKind
from typing import cast

if TYPE_CHECKING:
  from ..models.sms_registration_view import SmsRegistrationView





T = TypeVar("T", bound="SmsFoundSender")



@_attrs_define
class SmsFoundSender:
    """ 
        Attributes:
            countries (list[str]):
            imported (bool):
            kind (SmsFoundSenderKind):
            label (str):
            provider_ref (str):
            registration (SmsRegistrationView):
            throughput (float):
            value (str):
     """

    countries: list[str]
    imported: bool
    kind: SmsFoundSenderKind
    label: str
    provider_ref: str
    registration: SmsRegistrationView
    throughput: float
    value: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        countries = self.countries



        imported = self.imported

        kind = self.kind.value

        label = self.label

        provider_ref = self.provider_ref

        registration = self.registration.to_dict()

        throughput = self.throughput

        value = self.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "countries": countries,
            "imported": imported,
            "kind": kind,
            "label": label,
            "providerRef": provider_ref,
            "registration": registration,
            "throughput": throughput,
            "value": value,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sms_registration_view import SmsRegistrationView # noqa: PLC0415
        d = dict(src_dict)
        countries = cast(list[str], d.pop("countries"))


        imported = d.pop("imported")

        kind = SmsFoundSenderKind(d.pop("kind"))




        label = d.pop("label")

        provider_ref = d.pop("providerRef")

        registration = SmsRegistrationView.from_dict(d.pop("registration"))




        throughput = d.pop("throughput")

        value = d.pop("value")

        sms_found_sender = cls(
            countries=countries,
            imported=imported,
            kind=kind,
            label=label,
            provider_ref=provider_ref,
            registration=registration,
            throughput=throughput,
            value=value,
        )

        return sms_found_sender

