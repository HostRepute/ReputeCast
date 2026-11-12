from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_estimate_input_service import VerificationEstimateInputService
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.verification_target import VerificationTarget





T = TypeVar("T", bound="VerificationEstimateInput")



@_attrs_define
class VerificationEstimateInput:
    """ 
        Attributes:
            target (VerificationTarget):
            service (VerificationEstimateInputService | Unset):
            skip_recent (bool | Unset): Skip addresses checked in the last 90 days (default true)
     """

    target: VerificationTarget
    service: VerificationEstimateInputService | Unset = UNSET
    skip_recent: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_target import VerificationTarget # noqa: PLC0415
        target = self.target.to_dict()

        service: str | Unset = UNSET
        if not isinstance(self.service, Unset):
            service = self.service.value


        skip_recent = self.skip_recent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "target": target,
        })
        if service is not UNSET:
            field_dict["service"] = service
        if skip_recent is not UNSET:
            field_dict["skipRecent"] = skip_recent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_target import VerificationTarget # noqa: PLC0415
        d = dict(src_dict)
        target = VerificationTarget.from_dict(d.pop("target"))




        _service = d.pop("service", UNSET)
        service: VerificationEstimateInputService | Unset
        if isinstance(_service,  Unset):
            service = UNSET
        else:
            service = VerificationEstimateInputService(_service)




        skip_recent = d.pop("skipRecent", UNSET)

        verification_estimate_input = cls(
            target=target,
            service=service,
            skip_recent=skip_recent,
        )

        return verification_estimate_input

