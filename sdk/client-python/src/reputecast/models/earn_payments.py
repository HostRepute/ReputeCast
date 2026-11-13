from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.earn_payments_active import EarnPaymentsActive
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.earn_checkout_settings import EarnCheckoutSettings
  from ..models.earn_connection import EarnConnection
  from ..models.earn_permission_check import EarnPermissionCheck
  from ..models.earn_provider_card import EarnProviderCard
  from ..models.earn_unmapped_price import EarnUnmappedPrice





T = TypeVar("T", bound="EarnPayments")



@_attrs_define
class EarnPayments:
    """ 
        Attributes:
            active (EarnPaymentsActive):
            active_subscriptions (int):
            can_edit_prices (bool):
            can_manage (bool):
            checkout (EarnCheckoutSettings):
            helpers (list[str]):
            pause_months (list[int]):
            permissions (list[EarnPermissionCheck]):
            providers (list[EarnProviderCard]):
            test_members (int):
            unmapped (list[EarnUnmappedPrice]):
            webhook_events (list[str]):
            live (EarnConnection | Unset):
            test (EarnConnection | Unset):
     """

    active: EarnPaymentsActive
    active_subscriptions: int
    can_edit_prices: bool
    can_manage: bool
    checkout: EarnCheckoutSettings
    helpers: list[str]
    pause_months: list[int]
    permissions: list[EarnPermissionCheck]
    providers: list[EarnProviderCard]
    test_members: int
    unmapped: list[EarnUnmappedPrice]
    webhook_events: list[str]
    live: EarnConnection | Unset = UNSET
    test: EarnConnection | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.earn_checkout_settings import EarnCheckoutSettings # noqa: PLC0415
        from ..models.earn_connection import EarnConnection # noqa: PLC0415
        from ..models.earn_permission_check import EarnPermissionCheck # noqa: PLC0415
        from ..models.earn_provider_card import EarnProviderCard # noqa: PLC0415
        from ..models.earn_unmapped_price import EarnUnmappedPrice # noqa: PLC0415
        active = self.active.value

        active_subscriptions = self.active_subscriptions

        can_edit_prices = self.can_edit_prices

        can_manage = self.can_manage

        checkout = self.checkout.to_dict()

        helpers = self.helpers



        pause_months = self.pause_months



        permissions = []
        for permissions_item_data in self.permissions:
            permissions_item = permissions_item_data.to_dict()
            permissions.append(permissions_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        test_members = self.test_members

        unmapped = []
        for unmapped_item_data in self.unmapped:
            unmapped_item = unmapped_item_data.to_dict()
            unmapped.append(unmapped_item)



        webhook_events = self.webhook_events



        live: dict[str, Any] | Unset = UNSET
        if not isinstance(self.live, Unset):
            live = self.live.to_dict()

        test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.test, Unset):
            test = self.test.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "activeSubscriptions": active_subscriptions,
            "canEditPrices": can_edit_prices,
            "canManage": can_manage,
            "checkout": checkout,
            "helpers": helpers,
            "pauseMonths": pause_months,
            "permissions": permissions,
            "providers": providers,
            "testMembers": test_members,
            "unmapped": unmapped,
            "webhookEvents": webhook_events,
        })
        if live is not UNSET:
            field_dict["live"] = live
        if test is not UNSET:
            field_dict["test"] = test

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.earn_checkout_settings import EarnCheckoutSettings # noqa: PLC0415
        from ..models.earn_connection import EarnConnection # noqa: PLC0415
        from ..models.earn_permission_check import EarnPermissionCheck # noqa: PLC0415
        from ..models.earn_provider_card import EarnProviderCard # noqa: PLC0415
        from ..models.earn_unmapped_price import EarnUnmappedPrice # noqa: PLC0415
        d = dict(src_dict)
        active = EarnPaymentsActive(d.pop("active"))




        active_subscriptions = d.pop("activeSubscriptions")

        can_edit_prices = d.pop("canEditPrices")

        can_manage = d.pop("canManage")

        checkout = EarnCheckoutSettings.from_dict(d.pop("checkout"))




        helpers = cast(list[str], d.pop("helpers"))


        pause_months = cast(list[int], d.pop("pauseMonths"))


        permissions = []
        _permissions = d.pop("permissions")
        for permissions_item_data in (_permissions):
            permissions_item = EarnPermissionCheck.from_dict(permissions_item_data)



            permissions.append(permissions_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = EarnProviderCard.from_dict(providers_item_data)



            providers.append(providers_item)


        test_members = d.pop("testMembers")

        unmapped = []
        _unmapped = d.pop("unmapped")
        for unmapped_item_data in (_unmapped):
            unmapped_item = EarnUnmappedPrice.from_dict(unmapped_item_data)



            unmapped.append(unmapped_item)


        webhook_events = cast(list[str], d.pop("webhookEvents"))


        _live = d.pop("live", UNSET)
        live: EarnConnection | Unset
        if isinstance(_live,  Unset):
            live = UNSET
        else:
            live = EarnConnection.from_dict(_live)




        _test = d.pop("test", UNSET)
        test: EarnConnection | Unset
        if isinstance(_test,  Unset):
            test = UNSET
        else:
            test = EarnConnection.from_dict(_test)




        earn_payments = cls(
            active=active,
            active_subscriptions=active_subscriptions,
            can_edit_prices=can_edit_prices,
            can_manage=can_manage,
            checkout=checkout,
            helpers=helpers,
            pause_months=pause_months,
            permissions=permissions,
            providers=providers,
            test_members=test_members,
            unmapped=unmapped,
            webhook_events=webhook_events,
            live=live,
            test=test,
        )

        return earn_payments

