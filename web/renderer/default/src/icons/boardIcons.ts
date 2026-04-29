import type { LucideIcon } from 'lucide-react'
import {
  List, Bookmark, Pin, Gift, Cake, GraduationCap, Backpack, Dumbbell, FileText,
  Book, Archive, CreditCard, Wallet, Footprints, Utensils, Wine, Pill, Dna,
  Dog, Cat, Rabbit, Bird, Fish, PawPrint,
  House, Building, Landmark, Tent,
  Monitor, Laptop, Music, Gamepad2, Headphones, Code, Terminal,
  Leaf, Feather, Flame, Droplet, Snowflake, Sun, Moon,
  PersonStanding, Users, Baby,
  ShoppingBasket, ShoppingCart, ShoppingBag, Package,
  Trophy, Target, Volleyball, CircleDot,
  Plane, Map, Sailboat, Car, TrainFront, Anchor, Rocket,
  Briefcase, Settings, Scissors, Compass, Braces, Lightbulb, MessageCircle,
  TriangleAlert, Star, Heart, Circle, Triangle, Diamond, Square, Zap, Flag,
  Coffee, Beer, Pizza, Apple,
} from 'lucide-react'

export const BOARD_ICONS: Record<string, LucideIcon> = {
  list: List,
  bookmark: Bookmark,
  pin: Pin,
  gift: Gift,
  cake: Cake,
  'graduation-cap': GraduationCap,
  backpack: Backpack,
  dumbbell: Dumbbell,
  file: FileText,
  book: Book,
  archive: Archive,
  'credit-card': CreditCard,
  wallet: Wallet,
  run: Footprints,
  utensils: Utensils,
  wine: Wine,
  pill: Pill,
  dna: Dna,
  dog: Dog,
  cat: Cat,
  rabbit: Rabbit,
  bird: Bird,
  fish: Fish,
  paw: PawPrint,
  home: House,
  building: Building,
  landmark: Landmark,
  tent: Tent,
  monitor: Monitor,
  laptop: Laptop,
  music: Music,
  gamepad: Gamepad2,
  headphones: Headphones,
  code: Code,
  terminal: Terminal,
  leaf: Leaf,
  feather: Feather,
  flame: Flame,
  droplet: Droplet,
  snowflake: Snowflake,
  sun: Sun,
  moon: Moon,
  walk: PersonStanding,
  users: Users,
  baby: Baby,
  'shopping-basket': ShoppingBasket,
  'shopping-cart': ShoppingCart,
  'shopping-bag': ShoppingBag,
  package: Package,
  trophy: Trophy,
  target: Target,
  volleyball: Volleyball,
  'circle-dot': CircleDot,
  plane: Plane,
  map: Map,
  sailboat: Sailboat,
  car: Car,
  train: TrainFront,
  anchor: Anchor,
  rocket: Rocket,
  briefcase: Briefcase,
  settings: Settings,
  scissors: Scissors,
  compass: Compass,
  braces: Braces,
  lightbulb: Lightbulb,
  message: MessageCircle,
  alert: TriangleAlert,
  star: Star,
  heart: Heart,
  circle: Circle,
  triangle: Triangle,
  diamond: Diamond,
  square: Square,
  zap: Zap,
  flag: Flag,
  coffee: Coffee,
  beer: Beer,
  pizza: Pizza,
  apple: Apple,
}

export const BOARD_ICON_SLUGS: readonly string[] = Object.keys(BOARD_ICONS)
export const DEFAULT_ICON_SLUG = 'list'

export const BOARD_EMOJI_ICONS: readonly string[] = [
  '🚀', '✅', '📌', '📋', '📚', '📝',
  '💼', '🎯', '🎨', '🎵', '🎮', '🏠',
  '🍕', '☕️', '🌟', '❤️', '🔥', '💡',
  '🛒', '✈️', '🐱', '🐶', '🌳', '🌙',
  '☀️', '🎉', '💎', '🏆', '🎓', '🍎',
]

export function getBoardIcon(slug: string | undefined): LucideIcon | undefined {
  if (!slug) return undefined
  return BOARD_ICONS[slug]
}

// An icon value is treated as emoji/unicode text (back-compat) when:
//   - non-empty, AND
//   - not a known SVG slug, AND
//   - ≤4 chars (covers emoji + VS16 + ZWJ short sequences), AND
//   - contains no latin letters (slugs are [a-z0-9-]).
export function isEmojiIcon(icon: string | undefined): boolean {
  if (!icon) return false
  if (BOARD_ICONS[icon]) return false
  if (icon.length > 4) return false
  return !/[a-zA-Z]/.test(icon)
}
