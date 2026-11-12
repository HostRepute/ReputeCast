from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sender_defaults_reply_handling import SenderDefaultsReplyHandling
from typing import cast

if TYPE_CHECKING:
  from ..models.sender_tracking_option import SenderTrackingOption





T = TypeVar("T", bound="SenderDefaults")



@_attrs_define
class SenderDefaults:
    """ 
        Attributes:
            automation_sender_id (str):
            default_sender_id (str):
            footer_address (str):
            list_unsubscribe_mailto (bool):
            reply_handling (SenderDefaultsReplyHandling):
            reply_options (list[str]):
            reply_to (str):
            system_sender_id (str):
            tracking_domain (str):
            tracking_options (list[SenderTrackingOption]):
            transactional_sender_id (str):
            utm_links (bool):
     """

    automation_sender_id: str
    default_sender_id: str
    footer_address: str
    list_unsubscribe_mailto: bool
    reply_handling: SenderDefaultsReplyHandling
    reply_options: list[str]
    reply_to: str
    system_sender_id: str
    tracking_domain: str
    tracking_options: list[SenderTrackingOption]
    transactional_sender_id: str
    utm_links: bool





    def to_dict(self) -> dict[str, Any]:
        from ..models.sender_tracking_option import SenderTrackingOption # noqa: PLC0415
        automation_sender_id = self.automation_sender_id

        default_sender_id = self.default_sender_id

        footer_address = self.footer_address

        list_unsubscribe_mailto = self.list_unsubscribe_mailto

        reply_handling = self.reply_handling.value

        reply_options = self.reply_options



        reply_to = self.reply_to

        system_sender_id = self.system_sender_id

        tracking_domain = self.tracking_domain

        tracking_options = []
        for tracking_options_item_data in self.tracking_options:
            tracking_options_item = tracking_options_item_data.to_dict()
            tracking_options.append(tracking_options_item)



        transactional_sender_id = self.transactional_sender_id

        utm_links = self.utm_links


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "automationSenderId": automation_sender_id,
            "defaultSenderId": default_sender_id,
            "footerAddress": footer_address,
            "listUnsubscribeMailto": list_unsubscribe_mailto,
            "replyHandling": reply_handling,
            "replyOptions": reply_options,
            "replyTo": reply_to,
            "systemSenderId": system_sender_id,
            "trackingDomain": tracking_domain,
            "trackingOptions": tracking_options,
            "transactionalSenderId": transactional_sender_id,
            "utmLinks": utm_links,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.sender_tracking_option import SenderTrackingOption # noqa: PLC0415
        d = dict(src_dict)
        automation_sender_id = d.pop("automationSenderId")

        default_sender_id = d.pop("defaultSenderId")

        footer_address = d.pop("footerAddress")

        list_unsubscribe_mailto = d.pop("listUnsubscribeMailto")

        reply_handling = SenderDefaultsReplyHandling(d.pop("replyHandling"))




        reply_options = cast(list[str], d.pop("replyOptions"))


        reply_to = d.pop("replyTo")

        system_sender_id = d.pop("systemSenderId")

        tracking_domain = d.pop("trackingDomain")

        tracking_options = []
        _tracking_options = d.pop("trackingOptions")
        for tracking_options_item_data in (_tracking_options):
            tracking_options_item = SenderTrackingOption.from_dict(tracking_options_item_data)



            tracking_options.append(tracking_options_item)


        transactional_sender_id = d.pop("transactionalSenderId")

        utm_links = d.pop("utmLinks")

        sender_defaults = cls(
            automation_sender_id=automation_sender_id,
            default_sender_id=default_sender_id,
            footer_address=footer_address,
            list_unsubscribe_mailto=list_unsubscribe_mailto,
            reply_handling=reply_handling,
            reply_options=reply_options,
            reply_to=reply_to,
            system_sender_id=system_sender_id,
            tracking_domain=tracking_domain,
            tracking_options=tracking_options,
            transactional_sender_id=transactional_sender_id,
            utm_links=utm_links,
        )

        return sender_defaults

