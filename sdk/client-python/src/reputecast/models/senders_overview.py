from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.sender_defaults import SenderDefaults
  from ..models.sender_identity import SenderIdentity
  from ..models.sender_provider_column import SenderProviderColumn
  from ..models.sending_domain_row import SendingDomainRow





T = TypeVar("T", bound="SendersOverview")



@_attrs_define
class SendersOverview:
    """ 
        Attributes:
            can_manage (bool):
            defaults (SenderDefaults):
            domains (list[SendingDomainRow]):
            helpers (list[str]):
            providers (list[SenderProviderColumn]):
            senders (list[SenderIdentity]):
     """

    can_manage: bool
    defaults: SenderDefaults
    domains: list[SendingDomainRow]
    helpers: list[str]
    providers: list[SenderProviderColumn]
    senders: list[SenderIdentity]





    def to_dict(self) -> dict[str, Any]:
        from ..models.sender_defaults import SenderDefaults # noqa: PLC0415
        from ..models.sender_identity import SenderIdentity # noqa: PLC0415
        from ..models.sender_provider_column import SenderProviderColumn # noqa: PLC0415
        from ..models.sending_domain_row import SendingDomainRow # noqa: PLC0415
        can_manage = self.can_manage

        defaults = self.defaults.to_dict()

        domains = []
        for domains_item_data in self.domains:
            domains_item = domains_item_data.to_dict()
            domains.append(domains_item)



        helpers = self.helpers



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        senders = []
        for senders_item_data in self.senders:
            senders_item = senders_item_data.to_dict()
            senders.append(senders_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canManage": can_manage,
            "defaults": defaults,
            "domains": domains,
            "helpers": helpers,
            "providers": providers,
            "senders": senders,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sender_defaults import SenderDefaults # noqa: PLC0415
        from ..models.sender_identity import SenderIdentity # noqa: PLC0415
        from ..models.sender_provider_column import SenderProviderColumn # noqa: PLC0415
        from ..models.sending_domain_row import SendingDomainRow # noqa: PLC0415
        d = dict(src_dict)
        can_manage = d.pop("canManage")

        defaults = SenderDefaults.from_dict(d.pop("defaults"))




        domains = []
        _domains = d.pop("domains")
        for domains_item_data in (_domains):
            domains_item = SendingDomainRow.from_dict(domains_item_data)



            domains.append(domains_item)


        helpers = cast(list[str], d.pop("helpers"))


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = SenderProviderColumn.from_dict(providers_item_data)



            providers.append(providers_item)


        senders = []
        _senders = d.pop("senders")
        for senders_item_data in (_senders):
            senders_item = SenderIdentity.from_dict(senders_item_data)



            senders.append(senders_item)


        senders_overview = cls(
            can_manage=can_manage,
            defaults=defaults,
            domains=domains,
            helpers=helpers,
            providers=providers,
            senders=senders,
        )

        return senders_overview

