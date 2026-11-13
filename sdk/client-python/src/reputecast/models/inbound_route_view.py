from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.inbound_route_view_kind import InboundRouteViewKind
from ..models.inbound_route_view_reply_handling import InboundRouteViewReplyHandling
from ..models.inbound_route_view_status import InboundRouteViewStatus
from ..models.inbound_route_view_status_reason import InboundRouteViewStatusReason
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.inbound_address_view import InboundAddressView
  from ..models.inbound_mailbox_view import InboundMailboxView
  from ..models.inbound_provider_choice import InboundProviderChoice
  from ..models.inbound_provider_ref import InboundProviderRef
  from ..models.inbound_setup_view import InboundSetupView
  from ..models.inbound_test_view import InboundTestView





T = TypeVar("T", bound="InboundRouteView")



@_attrs_define
class InboundRouteView:
    """ 
        Attributes:
            addresses (InboundAddressView):
            can_manage (bool):
            choices (list[InboundProviderChoice]):
            domain (str):
            helpers (list[str]):
            last_test (InboundTestView):
            reply_handling (InboundRouteViewReplyHandling):
            setup (InboundSetupView):
            status (InboundRouteViewStatus):
            status_detail (str):
            suggestion (str):
            hook_url (str | Unset):
            kind (InboundRouteViewKind | Unset):
            last_received_at (datetime.datetime | Unset):
            mailbox (InboundMailboxView | Unset):
            provider (InboundProviderRef | Unset):
            set_up_at (datetime.datetime | Unset):
            status_reason (InboundRouteViewStatusReason | Unset):
     """

    addresses: InboundAddressView
    can_manage: bool
    choices: list[InboundProviderChoice]
    domain: str
    helpers: list[str]
    last_test: InboundTestView
    reply_handling: InboundRouteViewReplyHandling
    setup: InboundSetupView
    status: InboundRouteViewStatus
    status_detail: str
    suggestion: str
    hook_url: str | Unset = UNSET
    kind: InboundRouteViewKind | Unset = UNSET
    last_received_at: datetime.datetime | Unset = UNSET
    mailbox: InboundMailboxView | Unset = UNSET
    provider: InboundProviderRef | Unset = UNSET
    set_up_at: datetime.datetime | Unset = UNSET
    status_reason: InboundRouteViewStatusReason | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.inbound_address_view import InboundAddressView # noqa: PLC0415
        from ..models.inbound_mailbox_view import InboundMailboxView # noqa: PLC0415
        from ..models.inbound_provider_choice import InboundProviderChoice # noqa: PLC0415
        from ..models.inbound_provider_ref import InboundProviderRef # noqa: PLC0415
        from ..models.inbound_setup_view import InboundSetupView # noqa: PLC0415
        from ..models.inbound_test_view import InboundTestView # noqa: PLC0415
        addresses = self.addresses.to_dict()

        can_manage = self.can_manage

        choices = []
        for choices_item_data in self.choices:
            choices_item = choices_item_data.to_dict()
            choices.append(choices_item)



        domain = self.domain

        helpers = self.helpers



        last_test = self.last_test.to_dict()

        reply_handling = self.reply_handling.value

        setup = self.setup.to_dict()

        status = self.status.value

        status_detail = self.status_detail

        suggestion = self.suggestion

        hook_url = self.hook_url

        kind: str | Unset = UNSET
        if not isinstance(self.kind, Unset):
            kind = self.kind.value


        last_received_at: str | Unset = UNSET
        if not isinstance(self.last_received_at, Unset):
            last_received_at = self.last_received_at.isoformat()

        mailbox: dict[str, Any] | Unset = UNSET
        if not isinstance(self.mailbox, Unset):
            mailbox = self.mailbox.to_dict()

        provider: dict[str, Any] | Unset = UNSET
        if not isinstance(self.provider, Unset):
            provider = self.provider.to_dict()

        set_up_at: str | Unset = UNSET
        if not isinstance(self.set_up_at, Unset):
            set_up_at = self.set_up_at.isoformat()

        status_reason: str | Unset = UNSET
        if not isinstance(self.status_reason, Unset):
            status_reason = self.status_reason.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addresses": addresses,
            "canManage": can_manage,
            "choices": choices,
            "domain": domain,
            "helpers": helpers,
            "lastTest": last_test,
            "replyHandling": reply_handling,
            "setup": setup,
            "status": status,
            "statusDetail": status_detail,
            "suggestion": suggestion,
        })
        if hook_url is not UNSET:
            field_dict["hookUrl"] = hook_url
        if kind is not UNSET:
            field_dict["kind"] = kind
        if last_received_at is not UNSET:
            field_dict["lastReceivedAt"] = last_received_at
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if provider is not UNSET:
            field_dict["provider"] = provider
        if set_up_at is not UNSET:
            field_dict["setUpAt"] = set_up_at
        if status_reason is not UNSET:
            field_dict["statusReason"] = status_reason

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.inbound_address_view import InboundAddressView # noqa: PLC0415
        from ..models.inbound_mailbox_view import InboundMailboxView # noqa: PLC0415
        from ..models.inbound_provider_choice import InboundProviderChoice # noqa: PLC0415
        from ..models.inbound_provider_ref import InboundProviderRef # noqa: PLC0415
        from ..models.inbound_setup_view import InboundSetupView # noqa: PLC0415
        from ..models.inbound_test_view import InboundTestView # noqa: PLC0415
        d = dict(src_dict)
        addresses = InboundAddressView.from_dict(d.pop("addresses"))




        can_manage = d.pop("canManage")

        choices = []
        _choices = d.pop("choices")
        for choices_item_data in (_choices):
            choices_item = InboundProviderChoice.from_dict(choices_item_data)



            choices.append(choices_item)


        domain = d.pop("domain")

        helpers = cast(list[str], d.pop("helpers"))


        last_test = InboundTestView.from_dict(d.pop("lastTest"))




        reply_handling = InboundRouteViewReplyHandling(d.pop("replyHandling"))




        setup = InboundSetupView.from_dict(d.pop("setup"))




        status = InboundRouteViewStatus(d.pop("status"))




        status_detail = d.pop("statusDetail")

        suggestion = d.pop("suggestion")

        hook_url = d.pop("hookUrl", UNSET)

        _kind = d.pop("kind", UNSET)
        kind: InboundRouteViewKind | Unset
        if isinstance(_kind,  Unset):
            kind = UNSET
        else:
            kind = InboundRouteViewKind(_kind)




        _last_received_at = d.pop("lastReceivedAt", UNSET)
        last_received_at: datetime.datetime | Unset
        if isinstance(_last_received_at,  Unset):
            last_received_at = UNSET
        else:
            last_received_at = datetime.datetime.fromisoformat(_last_received_at)




        _mailbox = d.pop("mailbox", UNSET)
        mailbox: InboundMailboxView | Unset
        if isinstance(_mailbox,  Unset):
            mailbox = UNSET
        else:
            mailbox = InboundMailboxView.from_dict(_mailbox)




        _provider = d.pop("provider", UNSET)
        provider: InboundProviderRef | Unset
        if isinstance(_provider,  Unset):
            provider = UNSET
        else:
            provider = InboundProviderRef.from_dict(_provider)




        _set_up_at = d.pop("setUpAt", UNSET)
        set_up_at: datetime.datetime | Unset
        if isinstance(_set_up_at,  Unset):
            set_up_at = UNSET
        else:
            set_up_at = datetime.datetime.fromisoformat(_set_up_at)




        _status_reason = d.pop("statusReason", UNSET)
        status_reason: InboundRouteViewStatusReason | Unset
        if isinstance(_status_reason,  Unset):
            status_reason = UNSET
        else:
            status_reason = InboundRouteViewStatusReason(_status_reason)




        inbound_route_view = cls(
            addresses=addresses,
            can_manage=can_manage,
            choices=choices,
            domain=domain,
            helpers=helpers,
            last_test=last_test,
            reply_handling=reply_handling,
            setup=setup,
            status=status,
            status_detail=status_detail,
            suggestion=suggestion,
            hook_url=hook_url,
            kind=kind,
            last_received_at=last_received_at,
            mailbox=mailbox,
            provider=provider,
            set_up_at=set_up_at,
            status_reason=status_reason,
        )

        return inbound_route_view

