from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.verification_start_input_origin import VerificationStartInputOrigin
from ..models.verification_start_input_service import VerificationStartInputService
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.verification_target import VerificationTarget





T = TypeVar("T", bound="VerificationStartInput")



@_attrs_define
class VerificationStartInput:
    """ 
        Attributes:
            target (VerificationTarget):
            campaign_id (str | Unset):
            origin (VerificationStartInputOrigin | Unset):
            service (VerificationStartInputService | Unset):
            skip_recent (bool | Unset):
     """

    target: VerificationTarget
    campaign_id: str | Unset = UNSET
    origin: VerificationStartInputOrigin | Unset = UNSET
    service: VerificationStartInputService | Unset = UNSET
    skip_recent: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_target import VerificationTarget # noqa: PLC0415
        target = self.target.to_dict()

        campaign_id = self.campaign_id

        origin: str | Unset = UNSET
        if not isinstance(self.origin, Unset):
            origin = self.origin.value


        service: str | Unset = UNSET
        if not isinstance(self.service, Unset):
            service = self.service.value


        skip_recent = self.skip_recent


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "target": target,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if origin is not UNSET:
            field_dict["origin"] = origin
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




        campaign_id = d.pop("campaignId", UNSET)

        _origin = d.pop("origin", UNSET)
        origin: VerificationStartInputOrigin | Unset
        if isinstance(_origin,  Unset):
            origin = UNSET
        else:
            origin = VerificationStartInputOrigin(_origin)




        _service = d.pop("service", UNSET)
        service: VerificationStartInputService | Unset
        if isinstance(_service,  Unset):
            service = UNSET
        else:
            service = VerificationStartInputService(_service)




        skip_recent = d.pop("skipRecent", UNSET)

        verification_start_input = cls(
            target=target,
            campaign_id=campaign_id,
            origin=origin,
            service=service,
            skip_recent=skip_recent,
        )

        return verification_start_input

