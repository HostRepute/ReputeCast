from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.verification_service_view import VerificationServiceView
  from ..models.verification_use import VerificationUse





T = TypeVar("T", bound="VerificationOverview")



@_attrs_define
class VerificationOverview:
    """ 
        Attributes:
            can_manage (bool):
            can_verify (bool):
            manage_helpers (list[str]):
            services (list[VerificationServiceView]):
            uses (list[VerificationUse]):
            verify_helpers (list[str]):
            default_bulk (str | Unset):
            default_forms (str | Unset):
     """

    can_manage: bool
    can_verify: bool
    manage_helpers: list[str]
    services: list[VerificationServiceView]
    uses: list[VerificationUse]
    verify_helpers: list[str]
    default_bulk: str | Unset = UNSET
    default_forms: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.verification_service_view import VerificationServiceView # noqa: PLC0415
        from ..models.verification_use import VerificationUse # noqa: PLC0415
        can_manage = self.can_manage

        can_verify = self.can_verify

        manage_helpers = self.manage_helpers



        services = []
        for services_item_data in self.services:
            services_item = services_item_data.to_dict()
            services.append(services_item)



        uses = []
        for uses_item_data in self.uses:
            uses_item = uses_item_data.to_dict()
            uses.append(uses_item)



        verify_helpers = self.verify_helpers



        default_bulk = self.default_bulk

        default_forms = self.default_forms


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "canVerify": can_verify,
            "manageHelpers": manage_helpers,
            "services": services,
            "uses": uses,
            "verifyHelpers": verify_helpers,
        })
        if default_bulk is not UNSET:
            field_dict["defaultBulk"] = default_bulk
        if default_forms is not UNSET:
            field_dict["defaultForms"] = default_forms

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.verification_service_view import VerificationServiceView # noqa: PLC0415
        from ..models.verification_use import VerificationUse # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        can_verify = d.pop("canVerify")

        manage_helpers = cast(list[str], d.pop("manageHelpers"))


        services = []
        _services = d.pop("services")
        for services_item_data in (_services):
            services_item = VerificationServiceView.from_dict(services_item_data)



            services.append(services_item)


        uses = []
        _uses = d.pop("uses")
        for uses_item_data in (_uses):
            uses_item = VerificationUse.from_dict(uses_item_data)



            uses.append(uses_item)


        verify_helpers = cast(list[str], d.pop("verifyHelpers"))


        default_bulk = d.pop("defaultBulk", UNSET)

        default_forms = d.pop("defaultForms", UNSET)

        verification_overview = cls(
            can_manage=can_manage,
            can_verify=can_verify,
            manage_helpers=manage_helpers,
            services=services,
            uses=uses,
            verify_helpers=verify_helpers,
            default_bulk=default_bulk,
            default_forms=default_forms,
        )

        return verification_overview

