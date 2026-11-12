from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.automation_email_option import AutomationEmailOption
  from ..models.automation_field_option import AutomationFieldOption
  from ..models.automation_form_option import AutomationFormOption
  from ..models.automation_milestone_option import AutomationMilestoneOption
  from ..models.automation_option_item import AutomationOptionItem
  from ..models.automation_route_provider import AutomationRouteProvider
  from ..models.automation_segment_option import AutomationSegmentOption
  from ..models.automation_tier_option import AutomationTierOption
  from ..models.scoring_bands import ScoringBands





T = TypeVar("T", bound="AutomationOptions")



@_attrs_define
class AutomationOptions:
    """ 
        Attributes:
            campaigns (list[AutomationOptionItem]):
            emails (list[AutomationEmailOption]):
            events (list[str]):
            fields (list[AutomationFieldOption]):
            forms (list[AutomationFormOption]):
            lists (list[AutomationOptionItem]):
            milestones (list[AutomationMilestoneOption]):
            paid_connected (bool):
            products (list[AutomationOptionItem]):
            providers (list[AutomationRouteProvider]):
            segments (list[AutomationSegmentOption]):
            tags (list[str]):
            tiers (list[AutomationTierOption]):
            score_bands (ScoringBands | Unset):
     """

    campaigns: list[AutomationOptionItem]
    emails: list[AutomationEmailOption]
    events: list[str]
    fields: list[AutomationFieldOption]
    forms: list[AutomationFormOption]
    lists: list[AutomationOptionItem]
    milestones: list[AutomationMilestoneOption]
    paid_connected: bool
    products: list[AutomationOptionItem]
    providers: list[AutomationRouteProvider]
    segments: list[AutomationSegmentOption]
    tags: list[str]
    tiers: list[AutomationTierOption]
    score_bands: ScoringBands | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.automation_email_option import AutomationEmailOption # noqa: PLC0415
        from ..models.automation_field_option import AutomationFieldOption # noqa: PLC0415
        from ..models.automation_form_option import AutomationFormOption # noqa: PLC0415
        from ..models.automation_milestone_option import AutomationMilestoneOption # noqa: PLC0415
        from ..models.automation_option_item import AutomationOptionItem # noqa: PLC0415
        from ..models.automation_route_provider import AutomationRouteProvider # noqa: PLC0415
        from ..models.automation_segment_option import AutomationSegmentOption # noqa: PLC0415
        from ..models.automation_tier_option import AutomationTierOption # noqa: PLC0415
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        campaigns = []
        for campaigns_item_data in self.campaigns:
            campaigns_item = campaigns_item_data.to_dict()
            campaigns.append(campaigns_item)



        emails = []
        for emails_item_data in self.emails:
            emails_item = emails_item_data.to_dict()
            emails.append(emails_item)



        events = self.events



        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        forms = []
        for forms_item_data in self.forms:
            forms_item = forms_item_data.to_dict()
            forms.append(forms_item)



        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        milestones = []
        for milestones_item_data in self.milestones:
            milestones_item = milestones_item_data.to_dict()
            milestones.append(milestones_item)



        paid_connected = self.paid_connected

        products = []
        for products_item_data in self.products:
            products_item = products_item_data.to_dict()
            products.append(products_item)



        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        segments = []
        for segments_item_data in self.segments:
            segments_item = segments_item_data.to_dict()
            segments.append(segments_item)



        tags = self.tags



        tiers = []
        for tiers_item_data in self.tiers:
            tiers_item = tiers_item_data.to_dict()
            tiers.append(tiers_item)



        score_bands: dict[str, Any] | Unset = UNSET
        if not isinstance(self.score_bands, Unset):
            score_bands = self.score_bands.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "campaigns": campaigns,
            "emails": emails,
            "events": events,
            "fields": fields,
            "forms": forms,
            "lists": lists,
            "milestones": milestones,
            "paidConnected": paid_connected,
            "products": products,
            "providers": providers,
            "segments": segments,
            "tags": tags,
            "tiers": tiers,
        })
        if score_bands is not UNSET:
            field_dict["scoreBands"] = score_bands

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.automation_email_option import AutomationEmailOption # noqa: PLC0415
        from ..models.automation_field_option import AutomationFieldOption # noqa: PLC0415
        from ..models.automation_form_option import AutomationFormOption # noqa: PLC0415
        from ..models.automation_milestone_option import AutomationMilestoneOption # noqa: PLC0415
        from ..models.automation_option_item import AutomationOptionItem # noqa: PLC0415
        from ..models.automation_route_provider import AutomationRouteProvider # noqa: PLC0415
        from ..models.automation_segment_option import AutomationSegmentOption # noqa: PLC0415
        from ..models.automation_tier_option import AutomationTierOption # noqa: PLC0415
        from ..models.scoring_bands import ScoringBands # noqa: PLC0415
        d = dict(src_dict)
        campaigns = []
        _campaigns = d.pop("campaigns")
        for campaigns_item_data in (_campaigns):
            campaigns_item = AutomationOptionItem.from_dict(campaigns_item_data)



            campaigns.append(campaigns_item)


        emails = []
        _emails = d.pop("emails")
        for emails_item_data in (_emails):
            emails_item = AutomationEmailOption.from_dict(emails_item_data)



            emails.append(emails_item)


        events = cast(list[str], d.pop("events"))


        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = AutomationFieldOption.from_dict(fields_item_data)



            fields.append(fields_item)


        forms = []
        _forms = d.pop("forms")
        for forms_item_data in (_forms):
            forms_item = AutomationFormOption.from_dict(forms_item_data)



            forms.append(forms_item)


        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = AutomationOptionItem.from_dict(lists_item_data)



            lists.append(lists_item)


        milestones = []
        _milestones = d.pop("milestones")
        for milestones_item_data in (_milestones):
            milestones_item = AutomationMilestoneOption.from_dict(milestones_item_data)



            milestones.append(milestones_item)


        paid_connected = d.pop("paidConnected")

        products = []
        _products = d.pop("products")
        for products_item_data in (_products):
            products_item = AutomationOptionItem.from_dict(products_item_data)



            products.append(products_item)


        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = AutomationRouteProvider.from_dict(providers_item_data)



            providers.append(providers_item)


        segments = []
        _segments = d.pop("segments")
        for segments_item_data in (_segments):
            segments_item = AutomationSegmentOption.from_dict(segments_item_data)



            segments.append(segments_item)


        tags = cast(list[str], d.pop("tags"))


        tiers = []
        _tiers = d.pop("tiers")
        for tiers_item_data in (_tiers):
            tiers_item = AutomationTierOption.from_dict(tiers_item_data)



            tiers.append(tiers_item)


        _score_bands = d.pop("scoreBands", UNSET)
        score_bands: ScoringBands | Unset
        if isinstance(_score_bands,  Unset):
            score_bands = UNSET
        else:
            score_bands = ScoringBands.from_dict(_score_bands)




        automation_options = cls(
            campaigns=campaigns,
            emails=emails,
            events=events,
            fields=fields,
            forms=forms,
            lists=lists,
            milestones=milestones,
            paid_connected=paid_connected,
            products=products,
            providers=providers,
            segments=segments,
            tags=tags,
            tiers=tiers,
            score_bands=score_bands,
        )

        return automation_options

